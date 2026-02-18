package testutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Fixture constants for different language project files

// javaPomXML is a minimal Maven pom.xml for testing
const javaPomXML = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
    
    <name>Test Project</name>
    <description>Test project for OCI Build System</description>
    
    <properties>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>junit</groupId>
            <artifactId>junit</artifactId>
            <version>4.13.2</version>
            <scope>test</scope>
        </dependency>
    </dependencies>
    
    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-compiler-plugin</artifactId>
                <version>3.11.0</version>
            </plugin>
        </plugins>
    </build>
</project>
`

// dotnetCsproj is a minimal .NET project file for testing
const dotnetCsproj = `<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="Microsoft.Extensions.Logging" Version="8.0.0" />
  </ItemGroup>

</Project>
`

// goMod is a minimal Go module file for testing
const goMod = `module github.com/example/test-project

go 1.21

require (
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
)
`

// goSum is a minimal Go checksum file for testing
const goSum = `github.com/stretchr/testify v1.8.4 h1:CcVxjf3Q8PM0mHUKJCdn+eZZtm5yQwehR5yeSVQQcUk=
github.com/stretchr/testify v1.8.4/go.mod h1:sz/lmYIOXD/1dqDmKjjqLyZ2RngseejIcXlSw2iwfAo=
go.uber.org/zap v1.26.0 h1:sI7k6L95XOKS281NhVKOFCUNIvv9e0w4BF8N3u+tCRo=
go.uber.org/zap v1.26.0/go.mod h1:dtElttAiwGvoJ/vj4IwHBS/gXsEu/pZ50mUIRWuG0so=
`

// dockerfile is a minimal Dockerfile for testing
const dockerfile = `FROM alpine:latest

WORKDIR /app

COPY . .

CMD ["/bin/sh"]
`

// javaDockerfile is a Java-specific Dockerfile for testing
const javaDockerfile = `FROM maven:3.9-eclipse-temurin-17 AS builder

WORKDIR /app

COPY pom.xml .
RUN mvn dependency:go-offline

COPY src ./src
RUN mvn clean package -DskipTests

FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

COPY --from=builder /app/target/*.jar app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
`

// dotnetDockerfile is a .NET-specific Dockerfile for testing
const dotnetDockerfile = `FROM mcr.microsoft.com/dotnet/sdk:8.0 AS builder

WORKDIR /app

COPY *.csproj .
RUN dotnet restore

COPY . .
RUN dotnet publish -c Release -o out

FROM mcr.microsoft.com/dotnet/runtime:8.0

WORKDIR /app

COPY --from=builder /app/out .

ENTRYPOINT ["dotnet", "test-project.dll"]
`

// goDockerfile is a Go-specific Dockerfile for testing
const goDockerfile = `FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]
`

// CreateTempRepo creates a temporary repository for testing with the specified language
func CreateTempRepo(t *testing.T, language string) string {
	t.Helper()
	
	tmpDir := t.TempDir()
	
	switch language {
	case "java":
		createFile(t, filepath.Join(tmpDir, "pom.xml"), javaPomXML)
		createFile(t, filepath.Join(tmpDir, "Dockerfile"), javaDockerfile)
		
		// Create minimal Java source structure
		srcDir := filepath.Join(tmpDir, "src", "main", "java", "com", "example")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatalf("Failed to create Java source directory: %v", err)
		}
		createFile(t, filepath.Join(srcDir, "Main.java"), javaMainClass)
		
	case "dotnet":
		createFile(t, filepath.Join(tmpDir, "test-project.csproj"), dotnetCsproj)
		createFile(t, filepath.Join(tmpDir, "Dockerfile"), dotnetDockerfile)
		
		// Create minimal C# source
		createFile(t, filepath.Join(tmpDir, "Program.cs"), dotnetProgram)
		
	case "go":
		createFile(t, filepath.Join(tmpDir, "go.mod"), goMod)
		createFile(t, filepath.Join(tmpDir, "go.sum"), goSum)
		createFile(t, filepath.Join(tmpDir, "Dockerfile"), goDockerfile)
		
		// Create minimal Go source
		createFile(t, filepath.Join(tmpDir, "main.go"), goMain)
		
	default:
		// Generic repository with just a Dockerfile
		createFile(t, filepath.Join(tmpDir, "Dockerfile"), dockerfile)
	}
	
	return tmpDir
}

// createFile is a helper to create a file with content
func createFile(t *testing.T, path string, content string) {
	t.Helper()
	
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
}

// LoadWebhookPayload creates a GitHub webhook payload for testing
func LoadWebhookPayload(t *testing.T, repoName string) map[string]interface{} {
	t.Helper()
	
	payload := map[string]interface{}{
		"ref":   "refs/heads/main",
		"after": "abc123def456789012345678901234567890abcd",
		"repository": map[string]interface{}{
			"name":      repoName,
			"full_name": "test-owner/" + repoName,
			"clone_url": "https://github.com/test-owner/" + repoName + ".git",
			"owner": map[string]interface{}{
				"login": "test-owner",
			},
		},
		"head_commit": map[string]interface{}{
			"id":      "abc123def456789012345678901234567890abcd",
			"message": "Test commit message",
			"author": map[string]interface{}{
				"name":  "Test Author",
				"email": "test@example.com",
			},
		},
	}
	
	return payload
}

// LoadWebhookPayloadWithBranch creates a GitHub webhook payload with custom branch
func LoadWebhookPayloadWithBranch(t *testing.T, repoName, branch, commitHash string) map[string]interface{} {
	t.Helper()
	
	payload := map[string]interface{}{
		"ref":   "refs/heads/" + branch,
		"after": commitHash,
		"repository": map[string]interface{}{
			"name":      repoName,
			"full_name": "test-owner/" + repoName,
			"clone_url": "https://github.com/test-owner/" + repoName + ".git",
			"owner": map[string]interface{}{
				"login": "test-owner",
			},
		},
		"head_commit": map[string]interface{}{
			"id":      commitHash,
			"message": "Test commit on " + branch,
			"author": map[string]interface{}{
				"name":  "Test Author",
				"email": "test@example.com",
			},
		},
	}
	
	return payload
}

// GenerateHMACSignature generates an HMAC-SHA256 signature for webhook validation
func GenerateHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// GenerateHMACSignatureForPayload generates signature for a map payload
func GenerateHMACSignatureForPayload(payload map[string]interface{}, secret string) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return GenerateHMACSignature(payloadBytes, secret), nil
}

// Minimal source code templates

const javaMainClass = `package com.example;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello from OCI Build System Test!");
    }
}
`

const dotnetProgram = `using System;

namespace TestProject
{
    class Program
    {
        static void Main(string[] args)
        {
            Console.WriteLine("Hello from OCI Build System Test!");
        }
    }
}
`

const goMain = `package main

import "fmt"

func main() {
	fmt.Println("Hello from OCI Build System Test!")
}
`
