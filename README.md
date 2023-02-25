# chatbot

`chatbot` is a work-in-progress WhatsApp chatbot that uses GPT-3.5.

## Usage

1. Create an `.env` file with the following environment variables:

    ```text
    POSTGRES_USER="user"
    POSTGRES_PASSWORD="password"
    POSTGRES_DB="db"
    
    OPENAI_API_KEY="key"
    ```

2. Start Postgres service:

    ```sh
    docker compose up -d
    ```

3. Run the chatbot:

    ```sh
    go run ./cmd/chatbot -d --postgres="postgres://user:password@hostname:port/db"
    ```
