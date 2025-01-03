import { ChatOpenAI, OpenAIEmbeddings } from '@langchain/openai';

const { OPENAI_API_KEY } = process.env;

export function createMemoizedOpenAI(modelName: string = 'gpt-3.5-turbo') {
  let _gptModel: ChatOpenAI;
  return () => {
    if (!_gptModel) {
      _gptModel = new ChatOpenAI({
        openAIApiKey: OPENAI_API_KEY,
        temperature: 0,
        modelName,
      });
    }
    return _gptModel;
  };
}

export function createMemoizedEmbeddedAI(modelName: string = 'text-embedding-3-small') {
  let embeddingsInstance: OpenAIEmbeddings | null = null;
  return () => {
    if (!embeddingsInstance) {
      embeddingsInstance = new OpenAIEmbeddings({
        openAIApiKey: OPENAI_API_KEY,
        model: modelName,
      });
    }
    return embeddingsInstance;
  };
}
