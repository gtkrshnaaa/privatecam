# Development Conventions

This document outlines the standard coding practices and conventions for the Private Cam project. All developers and contributions must adhere to these guidelines.

## 1. Directory Structure Cleanliness
* Keep the workspace directory clean and free of temporary, unused, or dangling files.
* Do not commit local editor configurations, log files, or build artifacts. Keep files organized within their defined directories:
  * `firmware/` – ESP32-CAM program code.
  * `backend/` – Golang server files.
  * `frontend/` – Static UI monitoring dashboard files.

## 2. Backend (Golang) Conventions
* **Variable Declarations**: 
  * The short variable declaration syntax (`:=`) is strictly forbidden.
  * Use the standard `var` keyword for all variable declarations to improve readability and ensure explicit type or declaration styles.
  * Example:
    ```go
    // Forbidden
    // x := 10
    
    // Required
    var x int = 10
    ```
* Keep functions simple, focused, and well-structured.

## 3. Frontend Conventions
* **Styling (CSS)**:
  * Use pure, vanilla CSS only.
  * Third-party CSS frameworks (e.g., Tailwind CSS, Bootstrap, Bulma) are strictly prohibited.
  * Write clean, organized, and modular CSS selectors.
* **UI Components**:
  * Component libraries, UI kits, or specialized components (e.g., shadcn/ui, Radix UI) are strictly prohibited.
  * Write native HTML, CSS, and JS components to keep the codebase simple, light, and dependency-free.
