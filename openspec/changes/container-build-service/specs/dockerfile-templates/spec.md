## ADDED Requirements

### Requirement: Generate Dockerfile from template
The system SHALL generate a Dockerfile for each buildable project using a language-specific template, ignoring any Dockerfile present in the project repository.

#### Scenario: Project has existing Dockerfile
- **WHEN** a project directory contains a Dockerfile
- **THEN** the system SHALL ignore it and generate a new Dockerfile from the service's template

#### Scenario: Dockerfile generated for detected language
- **WHEN** language detection returns a supported language
- **THEN** the system SHALL produce a valid multi-stage Dockerfile using the corresponding template

### Requirement: Go Dockerfile template
The system SHALL provide a multi-stage Go Dockerfile template that builds the binary and produces a minimal runtime image.

#### Scenario: Go project build
- **WHEN** a Go project is being built
- **THEN** the generated Dockerfile SHALL use a golang base image for the build stage, copy the entire monorepo root (`COPY . .`) so that shared packages under `libs/` are available, compile the binary by referencing the project subpath (e.g., `go build -o /bin/<project> ./apps/<project>/...`), and copy the resulting binary to a distroless/static base image for the runtime stage

#### Scenario: Go project with shared monorepo dependencies
- **WHEN** a Go project at `apps/api` imports packages from `libs/shared`
- **THEN** the generated Dockerfile SHALL copy the full monorepo root so that all imported packages are available during compilation, and the build SHALL succeed without requiring those packages to be vendored inside the project directory

### Requirement: Java Dockerfile template
The system SHALL provide a multi-stage Java Dockerfile template supporting both Maven and Gradle builds.

#### Scenario: Maven project build
- **WHEN** a Java project with Maven is being built
- **THEN** the generated Dockerfile SHALL use a Maven base image for the build stage, copy the project directory (`COPY apps/<project>/ .`), run `mvn package -DskipTests`, and copy the resulting JAR to an Eclipse Temurin JRE runtime image

#### Scenario: Gradle project build
- **WHEN** a Java project with Gradle is being built
- **THEN** the generated Dockerfile SHALL use a Gradle base image for the build stage, copy the project directory (`COPY apps/<project>/ .`), run `gradle build -x test`, and copy the resulting JAR to an Eclipse Temurin JRE runtime image

### Requirement: .NET Dockerfile template
The system SHALL provide a multi-stage .NET Dockerfile template that restores, publishes, and runs on the ASP.NET runtime image.

#### Scenario: .NET project build
- **WHEN** a .NET project is being built
- **THEN** the generated Dockerfile SHALL use the .NET SDK image for the build stage, copy the project directory (`COPY apps/<project>/ .`), run `dotnet restore` and `dotnet publish`, and copy the output to an ASP.NET runtime image

### Requirement: Template variables
The system SHALL inject project-specific variables into Dockerfile templates including project name, project subpath relative to monorepo root, and build output artifact name.

#### Scenario: Variables resolved in template
- **WHEN** a Dockerfile is generated from a template
- **THEN** all template variables (project name, project subpath within monorepo, output artifact name) SHALL be resolved to actual values from the project context
