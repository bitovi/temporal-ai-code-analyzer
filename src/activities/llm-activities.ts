import { getS3Object } from './s3-activities';
import { createMemoizedOpenAI, createMemoizedEmbeddedAI } from './gpt'
import { chaosExists } from './chaos';

const getGPTModel = createMemoizedOpenAI();
const getEmbeddedInstance = createMemoizedEmbeddedAI();

export type GetEmbeddingDataInput = {
  bucket: string;
  key: string;
};

export type GetEmbeddingCountInput = {
  repository: string;
};

export interface GetRelatedDocumentsInput {
  repository: string;
  query: string;
  limit: number;
};

export type GetEmbeddingDataOutput = {
  key: string;
  embedding: number[];
};

export interface ChatCompletion {
  choices: string[];
}

export type InvokePromptInput = {
  query: string;
  relatedContent: string[];
};


/**
 * Fetches embedding for a given text using the OpenAI Embeddings API.
 * @param text - The input text to embed.
 * @returns An array of numbers representing the embedding.
 */
export async function fetchEmbedding(text: string): Promise<number[]> {
  const gptModel = getEmbeddedInstance()
  const response = await gptModel.embedQuery(text);

  return response
}

/**
 * Retrieves embedding data for a given S3 object.
 * @param input - The input containing S3 bucket and key.
 * @returns An object containing the key and its embedding.
 */
export async function getEmbeddingData(input: GetEmbeddingDataInput): Promise<GetEmbeddingDataOutput> {
  const { exists: aws } = await chaosExists({ key: "aws" })
  if (aws) {
    throw Error("error getting object from S3 -- AWS is totally down")
  }
  
  const bodyBuffer = await getS3Object(input);
  const bodyString = bodyBuffer.toString();
  
  const { exists: openai } = await chaosExists({ key: "openai" })
  if (openai) {
    throw Error("error fetching embeddings -- OpenAI Rate Limit Reached")
  }

  try {
    const embedding = await fetchEmbedding(bodyString);
    
    return {
      key: input.key,
      embedding: embedding,
    };
  } catch (error: any) {
    const errorMsg = error.message;
    if (errorMsg.includes('maximum context length')) {
      return {
        key: input.key,
        embedding: [],
      };
    }
    console.error(`Error in getEmbeddingData: ${errorMsg}`);
    throw error;
  }
}

/**
 * Fetches a chat completion from OpenAI's Chat API.
 * @param input - Array of [role, content] pairs.
 * @returns The chat completion response.
 */
export async function fetchCompletion(input: Array<[string, string]>): Promise<string> {  
  const messages = input.map(([role, content]) => ({ role, content }));
  
  const gptModel = getGPTModel()
  const response = await gptModel.invoke(messages)
  
  return response.content as string
}

/**
 * Invokes a prompt by constructing a chat completion request.
 * @param input - The input containing the query and related content.
 * @returns The content of the first choice's message.
 */
export async function invokePrompt(input: InvokePromptInput): Promise<string> {
  const prompt: Array<[string, string]> = [
    ["system", "You are a friendly, helpful software assistant. Your goal is to help users understand the code within a Git repository."],
    ["system", "You should respond in short paragraphs, using Markdown formatting for any blocks of code, separated with two newlines to keep your responses easily readable."],
    ["system", "Whenever possible, use code examples derived from the documentation provided."],
    ["system", "Here are the files from the Git repository that are relevant to the user's question: " + input.relatedContent.join("\n\n")],
    ["user", input.query],
  ];

  const { exists } = await chaosExists({ key: "openai" })
  if (exists) {
    throw Error("error fetching embeddings -- OpenAI Rate Limit Reached")
  }
  
  try {
    const completion = await fetchCompletion(prompt);
    if (!completion || completion === "") {
      throw new Error('No choices returned in chat completion');
    }
    return completion
  } catch (error: any) {
    console.error(`Error in invokePrompt: ${error.message}`);
    throw error;
  }
}