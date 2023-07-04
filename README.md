# Blitz

Blitz is a command-line tool written in Go for load testing web servers. It allows you to simulate concurrent requests to your server and measure its performance under heavy load. Blitz is designed to be simple and easy to use.

## Installation

To install Blitz, you need to have Go installed on your machine. Then, you can use the following command to install the tool:

```shell
go get github.com/startswithzed/blitz/cmd/blitz
```

Make sure to add `$GOPATH/bin` to your `$PATH` environment variable so that you can run the `blitz` command from anywhere.

## Usage

Blitz can be used with a request specification file to define the requests that will be sent during the load test. The request specification file is a JSON file that describes the requests to be made, including the URL, HTTP method, headers, and body.

Here's an example of a request specification file:

```json
[
  {
    "verb": "GET",
    "url": "https://api.example.com/users",
    "headers": {
      "Authorization": "Bearer token"
    }
  },
  {
    "verb": "POST",
    "url": "https://api.example.com/users",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "name": "John Doe",
      "email": "john.doe@example.com"
    }
  }
]
```

To start a load test with Blitz, run the following command:

```shell
blitz --req-spec /path/to/spec.json
```

The `--req-spec` flag is used to specify the path to the request specification file. Blitz will then start sending requests to the server based on the provided specifications.

You can customize the load test by using additional flags:

- `--duration` or `-d`: Duration of the test in minutes (default: 1 minute).
- `--num-clients` or `-c`: Number of concurrent clients sending requests to the server (default: 1).

For example, to run a load test for 5 minutes with 10 concurrent clients, you can use the following command:

```shell
blitz --req-spec /path/to/spec.json --duration 5m --num-clients 10
```

During the load test, Blitz will display a real-time dashboard showing the request and response statistics, including the request rate, response rate, average response time, and errors.

## Dashboard

The dashboard provides a visual representation of the load test progress and statistics. It shows the following information:

- Duration: The duration of the load test.
- Request Rate: The number of requests sent per second.
- Response Rate: The number of responses received per second.
- Average Response Time: The average time taken to receive a response.
- Max Response Time: The maximum time taken to receive a response.
- Min Response Time: The minimum time taken to receive a response.
- Errors: The number of errors encountered during the load test.

The dashboard is updated in real-time as the load test progresses.

## Contributing

Contributions to Blitz are welcome! If you find a bug, have a feature request, or want to contribute code, please open an issue or submit a pull request on [GitHub](https://github.com/startswithzed/blitz).
