## ADDED Requirements

### Requirement: Detect Go projects
The system SHALL identify a project as Go when a `go.mod` file exists in the project root directory.

#### Scenario: go.mod present
- **WHEN** the project directory contains a `go.mod` file
- **THEN** the system SHALL classify the project as Go

### Requirement: Detect Java projects
The system SHALL identify a project as Java when a `pom.xml`, `build.gradle`, or `build.gradle.kts` file exists in the project root directory.

#### Scenario: Maven project (pom.xml)
- **WHEN** the project directory contains a `pom.xml` file
- **THEN** the system SHALL classify the project as Java with Maven build tool

#### Scenario: Gradle project (build.gradle)
- **WHEN** the project directory contains a `build.gradle` or `build.gradle.kts` file
- **THEN** the system SHALL classify the project as Java with Gradle build tool

### Requirement: Detect .NET projects
The system SHALL identify a project as .NET when a `*.csproj` file exists in the project root directory.

#### Scenario: .csproj present
- **WHEN** the project directory contains one or more `.csproj` files
- **THEN** the system SHALL classify the project as .NET

### Requirement: Handle unknown languages
The system SHALL fail the build for projects where no supported language is detected.

#### Scenario: No marker files found
- **WHEN** the project directory does not contain any recognized language marker files
- **THEN** the system SHALL skip the build for that project, log a warning with the project name, and continue processing other projects

### Requirement: Resolve language ambiguity with priority
The system SHALL resolve ambiguity when multiple language marker files exist by using a fixed priority order: Go > Java > .NET.

#### Scenario: Multiple marker files present
- **WHEN** a project directory contains both `go.mod` and `pom.xml`
- **THEN** the system SHALL classify the project as Go (highest priority)
