Faza 1: Podstawowa struktura i przełącznik bezpieczeństwa (MVP)
Wklejaj te instrukcje krok po kroku do chatu Copilota w swoim środowisku IDE.

Krok 1: Inicjalizacja projektu i serwera HTTP

Prompt do Copilota:
Create a new Go project using the Gin Gonic web framework. Set up the basic project structure with main.go. Create a global boolean variable SecurityEnabled = false which will act as our security toggle. Set up a basic HTTP server running on port 8080 with a simple GET /ping route to verify the server is working.

Krok 2: Konfiguracja bazy danych SQLite

Prompt do Copilota:
Add SQLite database support using the github.com/mattn/go-sqlite3 driver and the standard database/sql package. Create a function InitDB(filepath string) *sql.DB that connects to a local SQLite file named app.db. If the database file does not exist, it should create it. Ensure the database connection is passed or accessible to the Gin router handlers.

Krok 3: Schemat bazy danych (Modele dla Bloga i Użytkowników)

Prompt do Copilota:
Write a function MigrateDB(db *sql.DB) that executes raw SQL to create two tables if they don't exist.
Table 1: blog with columns: id (INT Auto Increment Primary Key), title (VARCHAR 255), post_content (TEXT), published (TINYINT 1).
Table 2: users with columns: id (INT Auto Increment Primary Key), username (VARCHAR 50 UNIQUE), password_hash (VARCHAR 255), email (VARCHAR 100 UNIQUE).
Add a seed function to insert 2 dummy blog posts and 2 dummy users if the tables are empty.

Krok 4: Podstawowe endpointy (Szkielet)

Prompt do Copilota:
Create two basic Gin handlers:

GET /posts - fetches all published posts from the blog table and returns them as JSON.

POST /login - accepts JSON with username and password. For now, just return a success message if the username exists in the database (we will implement proper authentication later).
Remember to structure the handlers so that we can later easily add if SecurityEnabled branches inside them.
