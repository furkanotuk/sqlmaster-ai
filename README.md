
# SQLMaster-AI

**SQLMaster-AI** is a Go-based web service that generates optimized MySQL queries using OpenAI's GPT-4 Turbo model.  
It analyzes your database schema and creates SQL queries based on natural language prompts.

---

## Features

- Connects to a MySQL database and reads schema information (tables and columns).
- Uses OpenAI GPT-4 Turbo to generate SQL queries from user prompts.
- Provides a simple HTTP API endpoint (`/getsql`) that accepts POST requests with a prompt and returns the generated SQL.
- Handles CORS for easy integration with frontend applications.

---

## Getting Started

### Prerequisites

- Go 1.18 or higher
- MySQL database
- OpenAI API key

### Environment Variables

Create a `.env` file in the project root with the following variables:

```
DB_USER=your_mysql_user  
DB_PASS=your_mysql_password  
DB_HOST=localhost  
DB_PORT=3306  
DB_NAME=your_database_name  

OPENAI_API_KEY=your_openai_api_key
```

---

### Running the Server

```bash
go run main.go
```

The server will start and listen on the default port (e.g., 8080).

---

## API Usage

### POST `/getsql`

Generate a MySQL query based on your natural language prompt.

#### Request Body

```json
{
  "prompt": "Write a query to get all users created in the last month"
}
```

#### Response Body

```json
{
  "prompt": "Write a query to get all users created in the last month",
  "query": "SELECT * FROM users WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL 1 MONTH);"
}
```

---

## Notes

- Make sure your MySQL user has read access to the database schema.
- The service automatically reads your schema to provide context to GPT-4 for generating accurate queries.
- CORS is enabled to support frontend integrations.

---

## License

MIT License © [Furkan OTUK]

---

Feel free to contribute or open issues if you find bugs or want to request features!
