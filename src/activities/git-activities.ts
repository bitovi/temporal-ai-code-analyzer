import path from 'path';
import os from 'os';
import fs from 'fs/promises';
import child_process from 'node:child_process';
import { cleanRepository } from '../utils';
import { putS3Object } from './s3-activities';
import { chaosExists } from './chaos';

const configExtensions = ['.config', '.json', '.yaml', '.yml', '.ini', '.env', '.mod', '.sum'];
const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.bmp', '.svg', '.webp'];

export interface ArchiveRepositoryInput {
  repository: string;
  bucket: string;
}

export interface ArchiveRepositoryOutput {
  keys: string[];
}

function isHiddenFile(filePath: string): boolean {
  const parts = filePath.split(path.sep);
  return parts.some((part) => part.startsWith('.'));
}

function isConfigFile(filePath: string): boolean {
  return configExtensions.includes(path.extname(filePath).toLowerCase());
}

function isImageFile(filePath: string): boolean {
  return imageExtensions.includes(path.extname(filePath).toLowerCase());
}

async function walkDirectory(dir: string): Promise<string[]> {
  let files: string[] = [];

  const entries = await fs.readdir(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      const subFiles = await walkDirectory(fullPath);
      files = files.concat(subFiles);
    } else {
      files.push(fullPath);
    }
  }

  return files;
}

/**
 * Archives a git repository by cloning it, filtering files, and uploading to S3.
 * @param input - The repository URL and S3 bucket name.
 * @returns An object containing the list of uploaded S3 keys.
 */
export async function archiveRepository(
  input: ArchiveRepositoryInput
): Promise<ArchiveRepositoryOutput> {
  const temporaryDirectory = path.join(os.tmpdir(), cleanRepository(input.repository));
  const { exists } = await chaosExists({ key: 'github' });
  if (exists) {
    throw Error('error cloning repository -- are you sure you want to use GitHub?');
  }

  // Ensure the temporary directory is clean
  await fs.rm(temporaryDirectory, { recursive: true, force: true });
  console.log(`Removed temporary directory: ${temporaryDirectory}`);

  // Clone the repository
  child_process.execSync(`git clone --depth 1 ${input.repository} "${temporaryDirectory}"`);

  // Walk through the directory to get all file paths
  const allFiles = await walkDirectory(temporaryDirectory);
  console.log(`Total files found: ${allFiles.length}`);

  // Filter files
  const filteredFiles = allFiles.filter((filePath) => {
    return !isHiddenFile(filePath) && !isConfigFile(filePath) && !isImageFile(filePath);
  });
  console.log(`Files after filtering: ${filteredFiles.length}`);

  const keys: string[] = [];

  // Upload each file to S3
  for (const filePath of filteredFiles) {
    const fileData = await fs.readFile(filePath);
    const key = path.relative(temporaryDirectory, filePath).replace(/\\/g, '/');
    const { exists } = await chaosExists({ key: 'aws' });
    if (exists) {
      throw Error('error with S3.Put -- AWS is totally down');
    }
    await putS3Object({
      bucket: input.bucket,
      key,
      body: fileData,
    });
    keys.push(key);
    console.log(`Uploaded ${key} to S3 bucket ${input.bucket}.`);
  }

  return { keys: keys };
}
