import { proxyActivities } from '@temporalio/workflow';
import * as activities from './activities';
import { cleanRepository } from './utils';

const {
  createS3Bucket,
  deleteS3Object,
  deleteS3Bucket,
  archiveRepository,
  insertEmbedding,
  getRelatedDocuments,
  invokePrompt,
} = proxyActivities<typeof activities>({
  startToCloseTimeout: '1 minute',
  retry: {
    backoffCoefficient: 1,
    initialInterval: '3 seconds',
  },
});

const { getEmbeddingData } = proxyActivities<typeof activities>({
  startToCloseTimeout: '1 minute',
  retry: {
    initialInterval: '8 seconds',
    maximumAttempts: 5,
  },
});

export interface AnalyzeInput {
  repository: string;
  query: string;
}

export interface AnalyzeOutput {
  response: string;
}

/**
 * AnalyzeCode Workflow
 * @param input - AnalyzeInput containing repository and query
 * @returns AnalyzeOutput containing the response
 */
export async function analyzeCodeWorkflow(input: AnalyzeInput): Promise<AnalyzeOutput> {
  const bucketName = cleanRepository(input.repository);

  await createS3Bucket({ bucket: bucketName });

  const archiveResult = await archiveRepository({
    repository: input.repository,
    bucket: bucketName,
  });

  const embeddingPromises = archiveResult.keys.map((key: any) =>
    getEmbeddingData({ bucket: bucketName, key })
  );

  const embeddingResults = await Promise.all(embeddingPromises);
  const validEmbeddings = embeddingResults.filter(
    (e: { embedding: string | any[] }) => e.embedding.length > 0
  );

  const insertPromises = validEmbeddings.map((e: { key: any; embedding: any }) =>
    insertEmbedding({
      bucket: bucketName,
      repository: input.repository,
      key: e.key,
      embedding: e.embedding,
      content: '',
    })
  );

  await Promise.all(insertPromises);

  const deleteObjectPromises = archiveResult.keys.map((key: any) =>
    deleteS3Object({ bucket: bucketName, key })
  );

  await Promise.all(deleteObjectPromises);

  await deleteS3Bucket({ bucket: bucketName });

  const relatedDocuments = await getRelatedDocuments({
    repository: input.repository,
    query: input.query,
    limit: 5,
  });

  console.log({ relatedDocuments });

  const relatedContent = relatedDocuments.records.map((record: { content: any }) => record.content);

  const response = await invokePrompt({
    query: input.query,
    relatedContent,
  });

  return {
    response,
  };
}
