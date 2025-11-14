@echo off
REM Test runner script for Windows CMD
REM Usage: test.bat [command]

if "%1"=="" goto help
if "%1"=="help" goto help
if "%1"=="test" goto test
if "%1"=="test-verbose" goto test-verbose
if "%1"=="test-coverage" goto test-coverage
if "%1"=="test-coverage-html" goto test-coverage-html
if "%1"=="repo" goto repo
if "%1"=="event" goto event
if "%1"=="contact" goto contact
if "%1"=="file" goto file
if "%1"=="org" goto org
if "%1"=="user" goto user
if "%1"=="race" goto race
if "%1"=="build" goto build
if "%1"=="build-all" goto build-all
if "%1"=="clean" goto clean
goto unknown

:test
echo Running all tests...
go test ./...
goto end

:test-verbose
echo Running all tests (verbose)...
go test -v ./...
goto end

:test-coverage
echo Running tests with coverage...
go test -cover ./...
goto end

:test-coverage-html
echo Generating coverage report...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
echo Coverage report generated: coverage.html
start coverage.html
goto end

:repo
echo Running repository tests...
go test -v ./internal/storage/repo/...
goto end

:event
echo Running event repository tests...
go test -v -run TestEvent ./internal/storage/repo
goto end

:contact
echo Running contact repository tests...
go test -v -run TestContact ./internal/storage/repo
goto end

:file
echo Running file repository tests...
go test -v -run TestFile ./internal/storage/repo
goto end

:org
echo Running organization repository tests...
go test -v -run TestOrganization ./internal/storage/repo
goto end

:user
echo Running user repository tests...
go test -v -run TestUser ./internal/storage/repo
goto end

:race
echo Running tests with race detector...
go test -race ./...
goto end

:build
echo Building application...
go build -o bin/srmt-admin.exe ./cmd/srmt-admin
goto end

:build-all
echo Building all packages...
go build ./...
goto end

:clean
echo Cleaning...
go clean
go clean -testcache
if exist coverage.out del coverage.out
if exist coverage.html del coverage.html
if exist bin rmdir /s /q bin
echo Clean complete!
goto end

:help
echo Available commands:
echo   test                - Run all tests
echo   test-verbose        - Run tests with verbose output
echo   test-coverage       - Run tests with coverage report
echo   test-coverage-html  - Generate HTML coverage report
echo   repo                - Run only repository tests
echo   event               - Run only event repository tests
echo   contact             - Run only contact repository tests
echo   file                - Run only file repository tests
echo   org                 - Run only organization repository tests
echo   user                - Run only user repository tests
echo   race                - Run tests with race detector
echo   build               - Build the application
echo   build-all           - Build all packages
echo   clean               - Clean build artifacts and test cache
echo.
echo Example: test.bat repo
goto end

:unknown
echo Unknown command: %1
echo.
goto help

:end
