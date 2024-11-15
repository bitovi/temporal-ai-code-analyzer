# Temporal AI Code Analyzer

The following is a Generative AI application that allows an LLM to answer questions about any Git repository.

## Installing and running dependencies

This repo contains a simple local development setup. For production use, we would recommend using AWS.

Use the following command to run everything you need locally:

- Localstack (for storing files in local S3)
- Postgres (where embeddings are stored)
- A Temporal Worker (to run your Workflow/Activity code)

## Obtain an OpenAI API key

To run this project, you will need an OpenAI API key.  If you already have an OpenAI account, you can setup a project and API key [on the OpenAI Settings page](https://platform.openai.com/settings/). If you don't have an account, you can sign up at [OpenAI](https://platform.openai.com/signup).  You'll need to perform two main steps to run this project:

1. To create an API key, create a project, first, then open the API Keys page from the left sidebar and create a new key.
2. Open the `Limits` page for the new project and select the following models:
   - `gpt-3.5-turbo`
      - `text-embedding-ada-002`
         - `gpt-4-turbo`

         Once you've setup your API key and models, you'll be ready to run the project.  Note that sometimes it can take up to 15 minutes for the model selections to apply to your API key.

## Configure Environment Variables

You will need to set the OpenAI API Key as an environment variable and will also need environment variables set for connecting to Temporal Cloud. Create a `.env` file and fill it in:

```bash
cp .env-example .env
```


## Starting the application

```bash
./up.sh
```

See [these instructions](#obtain-an-openai-api-key) if you need an OpenAI key.

## Tearing everything down

Run the following command to turn everything off:

```bash
./down.sh
```

## Asking a question

The script below will start a new workflow:

```bash
go run src/client/main.go <Git Repo URL> <Question>
```

