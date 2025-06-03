# Property Testing Example

This repository demonstrates property-based testing in Go using a simple cache implementation. It accompanies the article [Property Testing: Finding Bugs You Didn't Know You Had](https://blog.platform.engineering/property-testing-finding-bugs-you-didnt-know-you-had-60bd4b5bc74c).

## What's Inside

- `cache.go`: A thread-safe cache implementation with expiration support
- `cache_test.go`: Traditional unit tests for the cache
- `cache_property_test.go`: Property-based tests demonstrating advanced testing concepts

## Properties Tested

1. **Basic Operations**: The cache maintains correct state after any sequence of operations
2. **Expiration**: Time-based behaviors work correctly
3. **LRU Eviction**: Least recently used items are evicted when size limit is reached

## Getting Started

1. Clone this repository:
```bash
git clone https://github.com/browdues/go-cache-property-tests
cd property-testing-example
```

2. Install dependencies:
```bash
go mod download
```

3. Run the tests:
```bash
# Run all tests
go test ./... -v

# Run only property tests
go test -run Property ./... -v

# Reproduce a failure
 go test -run <TestName> -rapid.seed=<seed-value>
```

## Learn More

- Read the accompanying [article](https://blog.platform.engineering/property-testing-finding-bugs-you-didnt-know-you-had-60bd4b5bc74c)
- Check out the [rapid testing framework](https://pkg.go.dev/pgregory.net/rapid)
