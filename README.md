# HTTP Load Testing Tool

[![Release](https://github.com/tklein1801/http-load-testing-tool/actions/workflows/release.yml/badge.svg)](https://github.com/tklein1801/http-load-testing-tool/actions/workflows/release.yml)

This tool allows you to send a large number of HTTP requests in parallel to a specified endpoint to test the performance and stability of your web application or API endpoint. It offers an easy way to assess the resilience of your server under high load conditions.

## Features

-Sending HTTP requests with configurable method (GET, POST, etc.)
-Parallelization of requests with an adjustable number of workers
-Ability to specify custom headers and query parameters
-Real-time progress display during testing
-Detailed summary of test results, including the number of successful/failed requests, total duration, data transferred, and requests per second

## Prerequisites

> [!NOTE]
> To use this tool, you need to have Go installed. The application was developed and tested with Go 1.15+.

## Installation

1. Clone the repository or download the source code:

   ```bash
   git clone https://github.com/tklein1801/http-load-testing-tool.git
   ```

2. Navigate to the tool's directory:

   ```bash
   cd http-load-testing-tool
   ```

3. Build the tool using go build:

   ```bash
   go build -o hltt
   ```

## Usage

After compiling, you can run the tool with various flags to configure your test:

```bash
./hltt -endpoint="https://yourapi.com/resource" -method=GET -amount=100 -worker=10 -output="results.json"
```

### Flags

- `-endpoint`: The URL endpoint to be tested.
- `-method`: The HTTP method for the requests (e.g., GET, POST). Default is GET.
- `-amount`: The total number of requests to send. Default is 1.
- `-worker`: The number of parallel workers (Go routines) to send requests. Default is 10.
- `-output`: The filename for outputting the test results in JSON format. Default is results.json.
- `-query`: Query parameters to append to the request. Format: key=value. Can be used multiple times.
- `-header`: Custom headers for the requests. Format: Key:Value. Can be used multiple times.

### Results

The test results are saved in a JSON file specified by the -output flag. The file contains detailed information about the test configuration and the results of each request.

## License

This tool is released under the MIT License. For more information, see the LICENSE file.
