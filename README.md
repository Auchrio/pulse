# Pulse

A decentralized, stateless messaging system for transmitting encrypted messages between systems using a shared ID. Pulse leverages Nostr relays to enable secure, peer-to-peer communication without requiring centralized servers or maintaining connection state.

## Features

- **Encrypted Messaging**: AES-256-GCM encryption with SHA256-derived keys
- **Decentralized**: Uses Nostr relay infrastructure for message distribution
- **Multiple Modes**: Send, retrieve, listen, and interactive chat modes
- **Configurable**: Customize relays, encryption keys, and more via `pulse.conf`
- **Verbose Mode**: Full relay status reporting with timing and error details
- **Cross-Platform**: Works on Windows, macOS, and Linux

## Installation

### From GitHub Releases

Download the latest precompiled binary from the [Auchrio/Pulse GitHub releases page](https://github.com/Auchrio/pulse/releases):

1. Visit https://github.com/Auchrio/pulse/releases
2. Download the binary for your platform:
   - `pulse.exe` - Windows
   - `pulse` - macOS/Linux
3. Extract the file to a location in your `PATH` or keep it in your working directory
4. (Optional) Generate a config file: `./pulse --generate-config`

### From Source

Requires Go 1.25 or later:

```bash
git clone https://github.com/Auchrio/pulse.git
cd pulse
go build -o pulse
```

## Quick Start

### Basic Message Operations

**Send a message:**
```bash
pulse myid "Hello, World!"
```

**Retrieve the most recent message:**
```bash
pulse myid
```

**Listen for incoming messages (30-second timeout by default):**
```bash
pulse myid -l <timeout>
```

**Interactive chat mode:**
```bash
pulse myid -c username
```

### Verbose Mode

Add the `-v` flag to any command to see detailed relay status:

```bash
pulse myid "Important message" -v
```

Output includes:
- Each relay connection status (✓ success, ✗ error/timeout)
- Response time from each relay
- Error messages (if any)
- Total operation time

Example:
```
Sending message...
[✓] wss://relay.damus.io           645ms (published)
[✓] wss://nos.lol                  420ms (published)
[✓] wss://relay.snort.social       459ms (published)
Total time: 420ms
Success
Total operation time: 654ms
```

## Usage Modes

### 1. Send Mode

Send an encrypted message to an ID:

```bash
pulse <id> "<message>" [options]
```

**Example:**
```bash
pulse alice "Meet me at the usual place" -v
```

- Encrypts the message using the ID as the encryption key
- Publishes to all configured relays
- Waits up to 10 seconds for at least one relay to accept the message
- Returns immediately upon first successful publish with `-v` showing timing

### 2. Retrieve Mode

Get the most recent message sent to an ID:

```bash
pulse <id> [options]
```

**Example:**
```bash
pulse alice -v
```

- Queries all configured relays for messages
- Waits up to 5 seconds for relay connections
- Collects responses for 300ms to ensure message freshness
- Returns the message with the most recent timestamp
- Outputs only the decrypted message (or error if none found)

### 3. Listen Mode

Listen for incoming messages on an ID with configurable timeout:

```bash
pulse <id> -l [options]
pulse <id> --listen [options]
```

**Examples:**
```bash
# Listen with default timeout (from config, default 30 seconds)
pulse alice -l

# Listen with custom 60-second timeout
pulse alice -l -t 60

# Listen with no timeout (waits indefinitely)
pulse alice -l -t 0

# Listen with verbose output
pulse alice -l -v
```

**Features:**
- Subscribes to all configured relays
- Waits for new messages arriving after the command starts
- Returns immediately when first message is received
- Ignores messages from the same session (deduplication)
- Timeout configurable via `-t` flag or `listen-timeout` in config

**Timeout Options:**
- No flag: Uses `listen-timeout` from `pulse.conf` (default: 30 seconds)
- `-t 0`: No timeout - waits indefinitely for a message
- `-t N`: Waits N seconds for a message
- `-t -1`: Uses config default (same as no flag)

### 4. Chat Mode

Interactive bidirectional chat on an ID:

```bash
pulse <id> -c [username] [options]
pulse <id> --chat [username] [options]
```

**Example:**
```bash
pulse alice-bob-channel -c Alice
```

- Loads message history from relays
- Displays previous messages
- Enters interactive prompt for sending new messages
- Shows incoming messages in real-time
- Auto-timestamps messages with `[HH:MM] Username: Message` format

**Features:**
- Message deduplication (doesn't show duplicates from multiple relays)
- Prevents echoing your own messages back
- Clean terminal interface with message refresh

## Configuration

Pulse can be configured via `pulse.conf` in the same directory as the executable.

### Generating Default Config

```bash
pulse --generate-config
# or
pulse -g
```

This creates `pulse.conf` with all settings and helpful comments.

### Configuration File Format

`pulse.conf` uses simple `key = value` format:

```properties
# List of Nostr relays (comma-separated)
relays = wss://relay.damus.io, wss://nos.lol, wss://relay.snort.social

# Maximum number of messages to retrieve from history
history-limit = 5

# Secret key for encryption (used with message ID to derive encryption key)
user-secret = super-secret-key

# Default username to use in chat mode (optional)
# If set, skips the username prompt unless overridden from command line
default-username = MyUsername

# Listen timeout in seconds (for -l flag, 0 = no timeout)
listen-timeout = 30
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `relays` | string (comma-separated) | `wss://relay.damus.io, wss://nos.lol, wss://relay.snort.social` | List of Nostr relay URLs to connect to |
| `history-limit` | int | `5` | Maximum number of messages to fetch from history |
| `user-secret` | string | `super-secret-key` | Master secret for deriving encryption keys (change this!) |
| `default-username` | string | (none) | Default username for chat mode (skips prompt if set) |
| `listen-timeout` | int | `30` | Default timeout in seconds for listen mode (0 = no timeout) |

**Note:** If `pulse.conf` doesn't exist, built-in defaults are used. Only override values you need to change.

## Command-Line Flags

```
Flags:
  -c, --chat              Enter chat mode
  -l, --listen            Listen for a new message
  -t, --listen-timeout    Listen timeout in seconds (0 = no timeout, -1 = use config default)
  -v, --verbose           Verbose output with relay status and timing
  -g, --generate-config   Generate pulse.conf with default settings
  -h, --help              Show help message
```

## Encryption & Security

Pulse uses industry-standard encryption:

- **Algorithm**: AES-256-GCM (NIST-approved Galois/Counter Mode)
- **Key Derivation**: SHA256(ID + UserSecret)
- **Nonce**: Randomly generated for each message
- **Tag Tagging**: Messages are tagged with the hashed key for relay filtering

**Important:**
- Change `user-secret` in `pulse.conf` to something unique
- Users sharing the same secret and ID can read each other's messages (this is intentional)
- The ID itself does NOT need to be secret
- Nostr messages are timestamped by relays

## How It Works

### Message Flow

1. **Sending**: `pulse id "message"` 
   - Derives encryption key from `id + user-secret`
   - Encrypts message with AES-256-GCM
   - Publishes to all configured Nostr relays
   - Returns when first relay accepts (or timeout)

2. **Retrieving**: `pulse id`
   - Queries all relays for messages tagged with hashed encryption key
   - Waits up to 300ms for all relays to respond
   - Selects message with most recent timestamp
   - Decrypts and returns

3. **Listening**: `pulse id -l`
   - Subscribes to relays from current time forward
   - Waits for new messages (30-second timeout)
   - Returns first message received

4. **Chatting**: `pulse id -c username`
   - Loads history, then subscribes for live updates
   - Forwards all user input to relays
   - Shows incoming messages in real-time

### Relay Interaction

- Each message is a Nostr event (Kind 1: Text Note)
- Messages include a tag with the hashed encryption key
- Messages include a timestamp set by the relay
- Messages are replicated across relays (with delays)

## Relay Configuration

Pulse comes pre-configured with reliable public Nostr relays. You can customize by editing `pulse.conf`:

**Popular Nostr Relays:**
- `wss://relay.damus.io` - Widely used, good uptime
- `wss://nos.lol` - Community relay, fast responses
- `wss://nostr.wine` - Stable, good performance
- `wss://nostr.band` - Good for discovery
- `wss://relay.snort.social` - Popular, reliable

## Examples

### Sending a Secure Message

```bash
$ pulse alice-workspace "Project deadline moved to Friday"
Success
```

### Checking for Updates

```bash
$ pulse system-notifications -v
Retrieving message...
[✓] wss://relay.damus.io           591ms (message retrieved)
[✓] wss://nos.lol                  404ms (message retrieved)
[✓] wss://relay.snort.social       493ms (message retrieved)
Total time: 404ms
System update available: v2.1.0
Total operation time: 591ms
```

### Interactive Chat

```bash
$ pulse team-channel -c Isaac
Loading history...
----- Previous Messages -----
[15:22] Bob: Meeting in 5 minutes
[15:24] Alice: On my way
--- Connected as [Isaac] ---
> Hi everyone!
[15:25] Isaac: Hi everyone!
[15:26] Bob: Hey Isaac!
> Thanks for the update
[15:26] Isaac: Thanks for the update
[15:27] Alice: You're welcome
```

### Listening for Events

```bash
$ pulse workflow-status -l -v
Retrieving message...
[✓] wss://relay.damus.io           145ms (message retrieved)
[✓] wss://nos.lol                  123ms (message retrieved)
Deployment successful: 3 services green
Total operation time: 201ms
```

## Troubleshooting

### No messages found

- **Cause**: Message hasn't propagated to queried relays yet
- **Solution**: Wait a few seconds and try again, or check relay connectivity with `-v` flag

### Relay connection errors

- **Cause**: Relay is down or unreachable
- **Solution**: Add more relays to `pulse.conf`, or check the relay's status page

### "context canceled" in verbose output

- **Cause**: This is normal - means operation succeeded on another relay and remaining connections were cancelled
- **Solution**: This is expected behavior, not an error

### Message decryption failed

- **Cause**: Wrong `user-secret` or corrupted message data
- **Solution**: Ensure all parties use the same `user-secret` value

## Performance Notes

- **Send**: ~200-700ms (depends on relay response times)
- **Retrieve**: ~300-600ms (waits 300ms for all relays to respond)
- **Listen**: ~100-300ms (depends on relay propagation)
- **Chat**: Real-time, limited by network latency

Retrieval waits for all relays to respond (up to 300ms) to ensure the most recent message is returned, even if a slower relay has a newer version.

## License

Refer to the LICENSE file in the repository.

## Contributing

Contributions welcome! Submit issues and pull requests at [github.com/Auchrio/pulse](https://github.com/Auchrio/pulse)

## Security Disclaimer

Pulse provides encryption in transit and at rest on relays. However:
- Relay operators can see the metadata (timestamps, IP addresses if over non-Tor)
- Message content is encrypted, but the fact that a message exists is visible
- Use multiple relays for redundancy but understand they're third-party services
-review the Nostr protocol documentation for security considerations
