# microvisor

A general purpose multi-kernel operating system.

## lazyos

A TUI to observe OS's on multiple machines over SSH.

### Run lazyos locally

#### Setup

**`/config.yml`**

machines:

- id : `<id>`
  label: `<label>`
  username: `<username>`
  hostname: `<hostname>`
  port: `<port>`
  add_keys_to_agent: `true/false`

**`/.env`**
PASSWORD\_`<id>`=`<password>` for each machine

#### Execution

1. $ `go build`

2. $ `./lazyos`
