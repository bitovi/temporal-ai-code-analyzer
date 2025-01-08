// src/main.ts

import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';
import dotenv from 'dotenv';
import { Connection, Client } from '@temporalio/client';
import { getTemporalClientOptions, cleanRepository } from '../utils';
import { analyzeCodeWorkflow, AnalyzeInput, AnalyzeOutput } from '../workflows';

dotenv.config();

interface Args {
  repository: string;
  query: string;
}

const argv = yargs(hideBin(process.argv))
  .usage('Usage: $0 <repository URL> <query>')
  .command('$0 <repository> <query>', 'Analyze a repository', (yargs) => {
    return yargs
      .positional('repository', {
        describe: 'URL of the repository to analyze',
        type: 'string',
        demandOption: true,
      })
      .positional('query', {
        describe: 'Query for the analysis',
        type: 'string',
        demandOption: true,
      });
  })
  .help()
  .alias('help', 'h')
  .parseSync() as unknown as Args;

async function main() {
  const { repository, query } = argv;

  console.log('Starting analysis for repository: %s with query: %s', repository, query);

  const connection = await Connection.connect(getTemporalClientOptions());

  const client = new Client({
    connection,
    namespace: process.env.TEMPORAL_NAMESPACE,
  });

  const input: AnalyzeInput = {
    repository: repository,
    query: query,
  };

  const workflowID = `analyze-${cleanRepository(repository)}`;

  const workflowOptions = {
    taskQueue: 'ai-code-analyzer-queue-ts',
    workflowId: workflowID,
  };

  try {
    const handle = await client.workflow.start(analyzeCodeWorkflow, {
      args: [input],
      ...workflowOptions,
    });

    console.log('Workflow started with ID: %s', handle.workflowId);

    const result: AnalyzeOutput = await handle.result();

    console.log(
      `Repository:\n${repository}\n\nQuestion:\n${query}\n\nResponse:\n${result.response}`
    );
  } catch (error: any) {
    console.error('Error executing workflow:', error);
    process.exit(1);
  }
}

main().catch((error) => {
  console.error('Unexpected error:', error);
  process.exit(1);
});
