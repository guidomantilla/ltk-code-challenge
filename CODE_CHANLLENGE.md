
Objective:
----------------
Create a public GIT repository and send the link back to the email.
add some unit test and brief doc


Assess your ability to design and implement a performant, idiomatic Golang RESTful API service with PostgreSQL integration, 
focusing on concurrency, JSON handling, and database interaction.

Challenge Description You are tasked with building a simple backend service in Go that manages a collection of "Events". 
Each Event has the following fields:

id (UUID, primary key)
title (string, max 100 characters)
description (string, optional)
start_time (timestamp)
end_time (timestamp)
created_at (timestamp, auto-set on creation)

Your service should expose a RESTful API with the following endpoints:

- Create Event: POST /events
Accepts a JSON payload with title, description, start_time, and end_time.
Validates that title is non-empty and <= 100 characters, start_time is before end_time.
Inserts the event into a PostgreSQL database, generating a UUID for id and setting created_at to current time.
Returns the created event as JSON with HTTP 201 status.

-List Events: GET /events
Returns a JSON array of all events ordered by start_time ascending.
Get Event by ID: GET /events/{id}
Returns the event with the specified UUID or 404 if not found.

Additional Requirements

- Use idiomatic Go, including proper error handling and concurrency-safe patterns.
- Use Go's database/sql package with the lib/pq driver or pgx for PostgreSQL interaction.
- Use JSON encoding/decoding with proper struct tags.
- Implement input validation and return appropriate HTTP status codes and error messages.
- Use context for request handling and database queries.

You do NOT need to implement authentication or Kafka integration for this challenge.
You can assume the PostgreSQL database is accessible and the events table is created with an appropriate schema.
    No Migrations!

Deliverables

- A single Go file or a small project that can be run locally.
- Instructions on how to run the service and test the endpoints (e.g., using curl or Postman).
- SQL schema for the events table.
- external_resources: | External Resources Required

PostgreSQL Database:
Use a local PostgreSQL instance or a free cloud-hosted PostgreSQL service such as:
- ElephantSQL Free Tier
- Supabase Free Tier
- Go Modules and Packages: