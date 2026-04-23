function ApiResult({ data, error }) {
  if (error) {
    return <pre className="result error">Error: {error}</pre>;
  }

  if (!data) {
    return <pre className="result muted">Run a request to see output.</pre>;
  }

  return <pre className="result">{JSON.stringify(data, null, 2)}</pre>;
}

export default ApiResult;
