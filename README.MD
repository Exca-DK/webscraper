# WEBSCRAPER

This is a simple example of a web scraper implemented using Golang, with zero external dependencies. This repository serves as a basic demonstration of how to create a web scraper to extract web content using the Golang standard library.


## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [Features](#features)

## Overview

A web scraper is a program that automatically extracts information from provided web pages. This example demonstrates how to create a basic web scraper using the Golang standard library.

Web scrapers are commonly used in tandem with web crawlers to feed the scraper with additional links, enabling the extraction of valuable data from a broader range of web sources.

## Getting Started

To get started with this example, follow these steps:

1. Clone this repository to your local machine:

   ```bash
   git clone https://github.com/Exca-DK/webscraper.git
   ```

2. Navigate to the project directory:

    ```bash
    cd webscraper
    ```

3. Explore and run:

    To run the web scraper help, execute the following command:

    ```bash
    go run main.go --help
    ```

    example configurations:
    ```bash
    go run main.go --urls=URL1,URL2
    go run main.go --urls=URL1,URL2 --threads=32
    go run main.go --urls=URL1,URL2 --threads=32 --verbosity=INFO
    ```

## Features

This example provides the following features:

- Efficient page content downloading.
- An adaptable cache system, which, by default, restricts revisiting websites for a specified lifetime, but can be configured to evict outdated entries.
- A built-in thread pool for managing and limiting concurrent tasks.
- A modular and extensible design for in-depth analysis of page content.
