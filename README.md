<div align="center">
  <h1>🚀 Algo Feedback System</h1>
  <p><strong>A comprehensive platform for managing students, courses, sessions, and automated feedback generation.</strong></p>
  
  [![Go Version](https://img.shields.io/github/go-mod/go-version/azharf99/algo-feedback?style=for-the-badge&logo=go)](https://go.dev/)
  [![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)](https://opensource.org/licenses/Apache-2.0)
  [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](http://makeapullrequest.com)
</div>

<br>

## 📖 Overview

Welcome to the **Algo Feedback System**! This is a robust backend service built with Golang, crafted specifically to streamline the management of educational programs. Whether you're handling student enrollments, organizing complex course structures, or needing automated, highly-customizable feedback generation, this system provides the foundation.

It offers specialized tools for tutors and administrators to track attendance, automatically generate beautiful PDF feedback reports, and seamlessly deliver them to students via WhatsApp integration.

---

## ✨ Key Features

- 👥 **Student & Group Management**: Intuitively manage students, cohorts, and their relationships with specific courses.
- 📚 **Course & Lesson Blueprints**: Define and structure study programs, modules, and individual lessons.
- 📅 **Session Tracking**: Schedule classes, manage meeting/recording links, and record precise student attendance.
- 🤖 **Automated Feedback System**:
  - **Feedback Seeder**: Automatically generates monthly feedback records based on real session history and attendance.
  - **Asynchronous PDF Generation**: Generates branded PDF reports using `maroto` with a built-in worker pool (task queue).
  - **WhatsApp Integration**: Schedules and sends feedback directly to users via a WhatsApp Gateway service.
- 🔄 **Batch Import via CSV**: Quickly upload CSV files to batch create or update records for Students, Courses, Groups, and Lessons using Upsert logic.
- 🔒 **Secure Authentication**: Robust role-based access control (Admin/Tutor) using JWT and Google OAuth2 support.
- 🌍 **Multi-language Support**: Built-in I18n middleware for internationalization.
- ⚡ **Performance**: Global rate limiting and optimized database queries using GORM.

---

## 📂 CSV Import Examples (`/examples`)

To make it easy to onboard and understand the batch import functionality, we have provided an **`examples`** folder in the root directory. 

This folder contains **all example CSV import file references** (`students_data.csv`, `courses_data.csv`, `groups_data.csv`, `lessons_data.csv`, `sessions_data.csv`) formatted exactly as the API expects. Use these templates to quickly populate your database via the batch import endpoints!

---

## 🛠️ Tech Stack

- **Language**: [Go (Golang)](https://go.dev/)
- **Framework**: [Gin Web Framework](https://gin-gonic.com/)
- **Database**: PostgreSQL with [GORM](https://gorm.io/)
- **PDF Generation**: [Maroto V2](https://github.com/johnfercher/maroto)
- **Authentication**: JWT & Google OAuth2
- **Task Queue**: Internal Worker Pool for background processing
- **Containerization**: Docker & Docker Compose

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) 1.22+ (optimized for performance)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Docker](https://docs.docker.com/get-docker/) (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/azharf99/algo-feedback.git
   cd algo-feedback
   ```

2. **Configure the Environment**
   Create a `.env` file in the root directory based on the environment variables used in `docker-compose.yml` or `cmd/api/main.go`.
   ```env
   DB_HOST=localhost
   DB_USER=postgres
   DB_PASSWORD=yourpassword
   DB_NAME=algo_feedback
   DB_PORT=5432
   JWT_SECRET=your_super_secret_key
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Run the Application**
   ```bash
   go run cmd/api/main.go
   ```
   *The server will start on `http://localhost:8080` by default.*

---

## 🔌 API Reference Highlights

- **Auth:** `POST /api/auth/login`, `POST /api/auth/google`
- **Feedback Automation:** 
  - `POST /api/feedbacks/seeder` - Batch generate feedback data
  - `POST /api/feedbacks/generate-pdf` - Trigger async PDF generation
  - `GET /api/feedbacks/:id/download` - Download generated reports
- **WhatsApp:** `POST /api/feedbacks/send-wa` - Dispatch reports via WhatsApp

---

## 🤝 Contributing

We welcome contributions! Please feel free to open an issue or submit a Pull Request.

---

## 📜 License & Attribution

This project is licensed under the **Apache License 2.0**.

**Mandatory Attribution:**
As per Section 4 of the Apache License 2.0, any redistribution or use of this software (or its derivatives) **MUST** include clear attribution to the original author: **Azhar Faturohman Ahidin**.

You must retain all copyright notices in the source code and include the `NOTICE` file in any distribution.

For more details, see the [LICENSE](LICENSE) and [NOTICE](NOTICE) files.

<br>
<div align="center">
  <i>Built with ❤️ by <a href="https://github.com/azharf99">Azhar Faturohman Ahidin</a>.</i>
</div>

