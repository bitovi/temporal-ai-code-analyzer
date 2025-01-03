import dotenv from 'dotenv';

dotenv.config();

const CHAOS_SERVER_URL = process.env.CHAOS_SERVER_URL;

export interface ChaosExistsInput {
  key: string;
}

export interface ChaosExistsOutput {
  exists: boolean;
}

/**
 * Checks if chaos exists for a given key by querying the Chaos server.
 * @param input - An object containing the key to check.
 * @returns An object indicating whether chaos exists.
 */
export async function chaosExists(input: ChaosExistsInput): Promise<ChaosExistsOutput> {
  const { key } = input;
  const url = `${CHAOS_SERVER_URL}?key=${encodeURIComponent(key)}`;

  try {
    console.log(`Sending GET request to ${url}`);
    const response = await fetch(url);

    const exists = response.status !== 200;
    console.log(`Received status code ${response.status} for key ${key}. Exists: ${exists}`);

    return { exists };
  } catch (error: any) {
    console.error(`Error fetching chaos status for key ${key}: ${error.message}`);
    return { exists: true };
  }
}
