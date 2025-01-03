import { Pool } from 'pg';
import { GetS3ObjectInput, getS3Object } from './s3-activities';
import { chaosExists } from './chaos';
import {
  fetchEmbedding,
  GetRelatedDocumentsInput,
  GetEmbeddingCountInput,
} from './llm-activities';
import pgvector from 'pgvector/pg';
import dotenv from 'dotenv';

dotenv.config();

export interface EmbeddingRecord {
  repository: string;
  key: string;
  content: string;
  embedding: number[];
};

export interface InsertEmbeddingInput extends EmbeddingRecord {
  bucket: string;
};

export type GetRelatedDocumentsOutput = {
  records: EmbeddingRecord[];
};

const pool = new Pool({
  connectionString: process.env.DATABASE_CONNECTION_STRING,
});

pool.on('connect', async function (client) {
  await pgvector.registerType(client);
});

pool.on('error', (err) => {
  console.error('Unexpected error on idle PostgreSQL client', err);
  process.exit(-1);
});

/**
 * Inserts an embedding record into the database.
 * @param input - The input data for insertion.
 */
export async function insertEmbedding(input: InsertEmbeddingInput): Promise<void> {
  const { exists } = await chaosExists({ key: "db" })
  if (exists) {
		throw Error("error inserting embedding -- DB out to lunch, back in 15 minutes")
	}
  const client = await pool.connect();
  try {
    const s3Input: GetS3ObjectInput = {
      bucket: input.bucket,
      key: input.key,
    };
    const content = await getS3Object(s3Input);

    const query = `
      INSERT INTO documents (repository, key, content, embedding)
      VALUES ($1, $2, $3, $4)
    `;
    const body = await content.Body?.transformToString();
    console.log('Content::', body)

    const values = [
      input.repository,
      input.key,
      body,
      pgvector.toSql(input.embedding),
    ];

    await client.query(query, values);
  } catch (error) {
    console.error('Error inserting embedding:', error);
    throw new Error(`Failed to insert embedding: ${(error as Error).message}`);
  } finally {
    client.release();
  }
}

/**
 * Retrieves the count of embeddings for a given repository.
 * @param input - The input data containing the repository name.
 * @returns The count of embeddings.
 */
export async function getEmbeddingCount(input: GetEmbeddingCountInput): Promise<number> {
  const { exists } = await chaosExists({ key: "db" })
  if (exists) {
		throw Error("error inserting embedding -- DB out to lunch, back in 15 minutes")
	}
  const client = await pool.connect();
  try {
    const query = 'SELECT COUNT(*) FROM documents WHERE repository = $1';
    const values = [input.repository];
    const res = await client.query(query, values);
    return parseInt(res.rows[0].count, 10);
  } catch (error) {
    console.error('Error fetching embedding count:', error);
    throw new Error(`Error fetching document count: ${(error as Error).message}`);
  } finally {
    client.release();
  }
}

/**
 * Retrieves related documents based on the query embedding.
 * @param input - The input data containing repository, query, and limit.
 * @returns An object containing an array of related embedding records.
 */
export async function getRelatedDocuments(input: GetRelatedDocumentsInput): Promise<GetRelatedDocumentsOutput> {
  const { exists } = await chaosExists({ key: "db" })
  if (exists) {
		throw Error("error inserting embedding -- DB out to lunch, back in 15 minutes")
	}
  try {
    const embeddingForQuery = await fetchEmbedding(input.query);

    const client = await pool.connect();
    try {
      const query = `
        SELECT key, content
        FROM documents
        WHERE repository = $1
        ORDER BY embedding <=> $2
        LIMIT $3
      `;
      const values = [
        input.repository,
        pgvector.toSql(embeddingForQuery),
        input.limit,
      ];

      const res = await client.query(query, values);

      const relatedRecords: EmbeddingRecord[] = res.rows.map((row: any) => ({
        repository: input.repository,
        key: row.key,
        content: row.content,
        embedding: row.embedding,
      }));

      return { records: relatedRecords };
    } catch (error) {
      console.error('Error fetching related documents:', error);
      throw new Error(`Error fetching related documents: ${(error as Error).message}`);
    } finally {
      client.release();
    }
  } catch (error) {
    console.error('Error in getRelatedDocuments:', error);
    throw error;
  }
}