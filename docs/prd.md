# Product Requirements Document: The Backend Problem Playground

## 1. Vision & Mission

**Vision:** To create an interactive, educational, and shareable platform that demonstrates complex backend engineering problems and their solutions in a hands-on, intuitive way.

**Mission:** To build a "playground" environment where users can actively trigger, observe, and understand common challenges in distributed systems, concurrency, and data management. This project will serve as a live portfolio piece, showcasing practical engineering skills beyond simple code repositories.

## 2. Target Audience

* **Technical Recruiters & Hiring Managers:** To quickly assess a candidate's practical problem-solving skills and depth of understanding.
* **Fellow Engineers & Tech Enthusiasts:** To learn about specific backend patterns and trade-offs in an interactive manner.
* **The Creator (Myself):** To consolidate, document, and deepen my own knowledge of the solutions I've implemented.

## 3. Core Concept: The "Scenario"

The fundamental building block of the Playground is the **Scenario**. Each Scenario is a self-contained demonstration of a specific engineering problem and its corresponding solution. The platform must be designed to easily accommodate new, diverse Scenarios without requiring architectural changes.

### 3.1. Scenario Anatomy (Required Components for each Scenario)

Every Scenario MUST consist of the following parts, which the platform's UI and backend will need to support generically.

* **`id`**: A unique machine-readable identifier (e.g., `cache_inconsistency`).
* **`title`**: A human-readable name (e.g., "DB/Cache Write Inconsistency").
* **`category`**: A high-level grouping (e.g., "Distributed", "MQ", "Concurrency").
* **`problem_description`**: A concise explanation of the problem. What is the business context? What goes wrong?
* **`solution_description`**: A concise explanation of the solution. How does it fix the problem?
* **`actions`**: A list of user-triggerable events. Each action represents an API call to the backend.
  * **Required Action:** At least one "problematic" action that demonstrates the issue.
  * **Required Action:** At least one "solution" action that demonstrates the fix.
  * Example Actions: `Update Product Price (Naive)`, `Process Payment (Idempotent)`.
* **`dashboard_-components`**: A definition of what state to display in the live dashboard. This tells the frontend which backend data points to visualize.
  * Examples: `State of MySQL Record`, `Content of Redis Key`, `Live Log Stream`, `Message Queue Length`.
* **`deep_dive_link`**: An optional URL to an external blog post or detailed `README` for further reading.

## 4. High-Level Functional Requirements

### 4.1. Frontend UI

The frontend should be a clean, single-page application (SPA).

* **FR1.1 - Scenario Navigator:** A sidebar or top menu that lists all available Scenarios, grouped by `category`. Clicking a Scenario loads it into the main view.
* **FR1.2 - Scenario Viewer:** The main content area, which dynamically renders the components of the selected Scenario.
  * Must display `title`, `problem_description`, and `solution_description`.
  * Must render a list of buttons for each defined `action`.
* **FR1.3 - Action Buttons:** Clicking an `action` button triggers a corresponding API call to the backend. The button should provide visual feedback (e.g., loading spinner) during the API call.
* **FR1.4 - Live Dashboard:** A dedicated section of the UI that visualizes the system's state in near real-time.
  * The Dashboard must be generic. It should render different "widgets" based on the `dashboard_components` defined by the current Scenario.
  * Example widgets: a simple key-value display for a DB record, a view for a Redis key, a scrolling log viewer.
  * The frontend will poll a backend endpoint (`/api/scenario/:id/state`) periodically to get the latest data for the dashboard.
* **FR1.5 - Deep Dive Link:** A clearly visible link or button that points to the `deep_dive_link`.

### 4.2. Backend API (Go)

The backend must be stateless and provide a generic API to support the frontend and the Scenario-based architecture.

* **FR2.1 - Scenario Configuration Endpoint:**
  * `GET /api/scenarios`: Returns a list of all available Scenarios (metadata only: `id`, `title`, `category`), allowing the frontend to build the navigator.
  * `GET /api/scenarios/:id`: Returns the full configuration for a single Scenario, including descriptions, actions, and dashboard definitions.
* **FR2.2 - Action Execution Endpoint:**
  * `POST /api/scenarios/:id/actions/:action_id`: A generic endpoint to execute any action for any scenario. The backend will map the `id` and `action_id` to the specific Go function to be executed.
* **FR2.3 - State Reporting Endpoint:**
  * `GET /api/scenarios/:id/state`: Returns a JSON object containing the current state of all `dashboard_components` for the given Scenario. This is the data source for the Frontend's Live Dashboard.

### 4.3. Infrastructure & Deployment

* **FR3.1 - One-Click Setup:** The entire environment (Frontend, Backend, MySQL, Redis, MQ, etc.) MUST be orchestrated via a single `docker-compose.yml` file for easy setup.
* **FR3.2 - Extensibility:** Adding a new Scenario should primarily involve:
    1. Creating a new Go package under a `scenarios/` directory.
    2. Implementing the required problem/solution logic.
    3. Defining the Scenario's configuration (title, actions, etc.) in a central registry.
    4. No changes to the frontend, core backend API, or infrastructure should be necessary.

## 5. Non-Functional Requirements

* **NFR1 - Performance:** The UI should be responsive. API responses and dashboard updates should be near real-time to provide a smooth user experience.
* **NFR2 - Clarity:** The purpose of each Scenario and the effect of each action should be immediately obvious to the user.
* **NFR3 - Simplicity:** The UI and overall design should be minimalist, focusing the user's attention on the technical demonstration. Avoid unnecessary visual clutter.
