# mcp-go-example

Model Context Protocol sqlite example server using Go.
This is just for learning purpose. No security implications, No multiuser support.

Python version is here.
https://github.com/modelcontextprotocol/servers/tree/main/src/sqlite

Python version quickstart is here.
https://modelcontextprotocol.io/quickstart


## Usasge

Edit Claude Desktop config at `~/Library/Application\ Support/Claude/claude_desktop_config.json`
```
{
  "mcpServers": {
    "sqlite": {
      "command": "path/to/mcp-go-example",
       "args": []
    }
  }
}
```

build and follow [quickstart](https://modelcontextprotocol.io/quickstart).


## Tasks

### test
```
go test
```

### build
Requires: test
```
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath .
```

### inspect
Requires: build
```
npx -y @modelcontextprotocol/inspector ./mcp-go-example 
```

### log
```
tail -f /tmp/mcp-go-example.log
```

## License

[MIT License](LICENSE)

Copyright (c) 2024 Masa Cento
