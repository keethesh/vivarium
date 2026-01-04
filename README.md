# 🕷️ VIVARIUM

> **"The ecosystem is the weapon. Resistance is organic failure."**

VIVARIUM is a modern, high-performance network stress testing toolkit built with a biological "Hive" architecture. It treats distributed computing as a living, breathing swarm.

**⚠️ WARNING: This tool is for educational purposes and authorized testing only.**

## 🚀 Quick Start

```bash
# Build
go build -o vivarium.exe ./cmd/vivarium

# See available commands
./vivarium --help

# Run a basic stress test (requires permission flag)
./vivarium sting locust --target https://your-target.com --rounds 100 --i-have-permission
```

## 🏛️ Architecture: The Hive Mind

| Concept | Hive Name | Description |
|---------|-----------|-------------|
| Your Instance | **The Queen** | The central intelligence coordinating all swarm activity |
| Web GUI | **The Hexagon** | Visual command interface (Phase 2) |
| Bots/Zombies | **Workers** | Open redirect URLs that amplify your attacks |
| Bot List Files | **The Comb** | Stored worker addresses, organized like honeycomb cells |
| Tor/Anonymity | **Smoke** | Anonymity layer (Phase 2) |

## ⚔️ Arsenal

### Stings (DoS - Single-Origin Attacks)

Precision strikes from the Queen herself:

```bash
# Locust - HTTP GET flood
vivarium sting locust --target <url> --rounds 1000 --concurrency 100 --i-have-permission

# Tick - Slowloris connection drain
vivarium sting tick --target <url> --sockets 200 --delay 15s --i-have-permission

# Fly Swarm - UDP flood
vivarium sting flyswarm --target <ip> --port 80 --rounds 10000 --i-have-permission
```

### Swarm (DDoS - Distributed via Open Redirects)

Mobilize the full colony through worker nodes:

```bash
# First, create a worker list (comb)
vivarium forage --output workers.txt

# Validate which workers actually work
vivarium comb validate --input workers.txt --output valid.txt

# Launch distributed attack through workers
vivarium swarm locust --target <url> --comb valid.txt --rounds 100 --i-have-permission
```

## 🔭 Senses (Reconnaissance)

### Scout - Asset Discovery

Find the largest resources on a target (best attack vectors):

```bash
vivarium sense scout --target https://example.com --depth 2 --top 10 --i-have-permission
```

## 🐝 Foraging (Worker Discovery)

Discover open redirect URLs via DuckDuckGo dorking:

```bash
# Use default dorks
vivarium forage --output discovered.txt -v

# Use custom dorks
vivarium forage --dorks my-dorks.txt --output discovered.txt
```

## 📦 Comb Management

Manage your worker lists:

```bash
# List workers in a comb file
vivarium comb list --file workers.txt

# Validate which URLs actually work as redirects
vivarium comb validate --input workers.txt --output valid.txt --concurrency 50

# Merge multiple comb files
vivarium comb merge file1.txt file2.txt --output merged.txt
```

## 🔧 Installation

### Requirements
- Go 1.25+

### Build from Source

```bash
git clone https://github.com/your-username/vivarium.git
cd vivarium
go build -o vivarium ./cmd/vivarium
```

### Create Alias (PowerShell)

Add to your `$PROFILE`:

```powershell
Set-Alias -Name viv -Value "C:\path\to\vivarium.exe"
```

## 📖 Command Reference

| Command | Description |
|---------|-------------|
| `sting locust` | HTTP GET flood attack |
| `sting tick` | Slowloris connection drain |
| `sting flyswarm` | UDP flood attack |
| `swarm locust` | Distributed HTTP flood via open redirects |
| `sense scout` | Discover largest assets on target |
| `forage` | Find open redirect URLs via dorking |
| `comb list` | Display workers in a comb file |
| `comb validate` | Test and filter working redirects |
| `comb merge` | Combine multiple comb files |

## 🛡️ Ethical Use

This tool requires the `--i-have-permission` flag for all attack operations. This is a reminder that you must:

1. **Only test systems you own** or have explicit written permission to test
2. **Never use this tool for malicious purposes**
3. **Understand the legal implications** in your jurisdiction

Using this tool without authorization may be illegal and is strictly prohibited.

## 📈 Roadmap

- [x] Phase 1: Core foundation
- [x] Phase 2: Sting attacks (Locust, Tick, FlySwarm)
- [x] Phase 3: Comb worker management
- [x] Phase 4: Swarm distributed attacks
- [x] Phase 5: Scout reconnaissance
- [x] Phase 6: Forage dorking
- [ ] Phase 7: Smoke (Tor/I2P integration)
- [ ] Phase 8: Hexagon GUI

---

*VIVARIUM is a tool for security research and stress testing. The colony does not care for borders, but its keepers must.*
