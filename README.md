<div align="center">
  <h1>🚀 Algo Feedback System</h1>
  <p><strong>A comprehensive platform for managing students, courses, sessions, and automated feedback generation.</strong></p>
  
  [![Go Version](https://img.shields.io/github/go-mod/go-version/azharf99/algo-feedback?style=for-the-badge&logo=go)](https://go.dev/)
  [![License: CC BY-NC-ND 4.0](https://img.shields.io/badge/License-CC%20BY--NC--ND%204.0-lightgrey.svg?style=for-the-badge)](https://creativecommons.org/licenses/by-nc-nd/4.0/)
  [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=for-the-badge)](http://makeapullrequest.com)
</div>

<br>

## 📖 Overview

Welcome to the **Algo Feedback System**! This is a robust backend service built with Golang, crafted specifically to streamline the management of educational programs. Whether you're handling student enrollments, organizing complex course structures, or needing automated, highly-customizable feedback generation, this system provides the foundation.

It offers specialized tools for tutors and administrators to track attendance, automatically generate beautiful PDF feedback reports, and seamlessly deliver them to parents or students via WhatsApp integration.

---

## ✨ Key Features

- 👥 **Student & Group Management**: Intuitively manage students, cohorts, and their relationships with specific courses.
- 📚 **Course & Lesson Blueprints**: Define and structure study programs, modules, and individual lessons.
- 📅 **Session Tracking**: Schedule classes, manage meeting/recording links, and record precise student attendance.
- 🤖 **Automated Feedback System**:
  - **Feedback Seeder**: Automatically generates monthly feedback records based on real session history and attendance.
  - **PDF Generation**: Asynchronously generates beautiful, branded PDF reports for students using `maroto`.
  - **WhatsApp Integration**: Schedules and sends feedback directly to users via WhatsApp (I use my own WhatsApp Gateway, but you can use any WhatsApp Gateway service).
- 🔄 **Batch Import via CSV**: Quickly upload CSV files to batch create or update records for Students, Courses, Groups, and Lessons using Upsert logic.
- 🔒 **Secure Authentication**: Robust role-based access control (Admin/Tutor) using JWT.

---

## 📂 CSV Import Examples (`/examples`)

To make it easy to onboard and understand the batch import functionality, we have provided an **`examples`** folder in the root directory. 

This folder contains **all example CSV import file references** (`students_data.csv`, `courses_data.csv`, `groups_data.csv`, `lessons_data.csv`, `sessions_data.csv`) formatted exactly as the API expects. Use these templates to quickly populate your database via the batch import endpoints!

---

## 🛠️ Tech Stack

- **Language**: [Go (Golang)](https://go.dev/) 1.26
- **Framework**: [Gin Web Framework](https://gin-gonic.com/)
- **Database**: PostgreSQL with [GORM](https://gorm.io/)
- **PDF Generation**: [Maroto](https://github.com/johnfercher/maroto)
- **Authentication**: JWT (JSON Web Tokens)
- **Containerization**: Docker & Docker Compose

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) 1.26 or higher
- [PostgreSQL](https://www.postgresql.org/download/)
- [Docker](https://docs.docker.com/get-docker/) (optional, for easy containerized setup)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/azharf99/algo-feedback.git
   cd algo-feedback
   ```

2. **Configure the Environment**
   Create a `.env` file in the root directory and fill in the required variables. Use the provided environment template as a reference:
   ```env
   # Database Configuration
   DB_HOST=localhost
   DB_USER=postgres
   DB_PASSWORD=yourpassword
   DB_NAME=algo_feedback
   DB_PORT=5432
   
   # JWT Configuration
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

### Using Docker

You can easily spin up the application and the PostgreSQL database using Docker Compose for a seamless development experience:

```bash
docker-compose up --build
```

---

## 🔌 API Reference Highlights

The API operates over REST and consumes/produces `application/json` (except for CSV imports which use `multipart/form-data`).

- **Auth:** `POST /api/auth/login`
- **Batch Imports:** `POST /api/students/import` (and courses, groups, lessons)
- **Attendance:** `POST /api/sessions/:id/attendance`
- **Feedback Automation:** 
  - `POST /api/feedbacks/seeder`
  - `POST /api/feedbacks/generate-pdf`

*(Please refer to `frontend-implementation.md` for a comprehensive API guide.)*

---

## 🤝 Contributing

We welcome contributions from the open-source community! Whether it's a bug report, a new feature, or documentation improvements, please feel free to open an issue or submit a Pull Request. Let's build a better educational tool together!

---

## 📞 Contact & Support

If you have any questions, suggestions, or want to discuss the project, feel free to reach out to me directly!

- **Email**: [azharfaturohman29@gmail.com](mailto:azharfaturohman29@gmail.com)
- **Telegram**: [@azhar_faturohman](https://t.me/azhar_faturohman)

---

## 📜 License & Copyright

This project is protected and licensed under the **Creative Commons Attribution-NonCommercial-NoDerivatives 4.0 International (CC BY-NC-ND 4.0)** to protect copyright and regulate distribution.

You are free to:
- **Share** — copy and redistribute the material in any medium or format.

Under the following terms:
- **Attribution** — You must give appropriate credit, provide a link to the license, and indicate if changes were made.
- **NonCommercial** — You may not use the material for commercial purposes.
- **NoDerivatives** — If you remix, transform, or build upon the material, you may not distribute the modified material.

For more details, please see the [LICENSE](LICENSE) file.

<br>
<div align="center">
  <i>Built with ❤️ for the open-source education community by Azhar Faturohman.</i>
</div>
