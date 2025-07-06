# 🚀 Log Monitoring Sidecar Setup Guide

This guide will help you set up the `gswarm-sidecar` repository to monitor and forward logs from your node to the central API.

---

## 1. Clone the Repository

```sh
git clone https://github.com/Deep-Commit/gswarm-sidecar.git
cd gswarm-sidecar
```

---

## 2. Install Go

Ensure you have Go installed (version 1.18+ recommended).

Check with:

```sh
go version
```

If you don't have Go, [download and install it here](https://go.dev/dl/).

---

## 3. Configure Log Monitoring

Edit the configuration file at `configs/config.yaml`.

Below is a sample configuration for log monitoring:

```yaml
log_monitoring:
  api_endpoint: "https://h9oy4hruxf.execute-api.us-east-1.amazonaws.com/prod/v1/ingest"
  batch_size: 10
  batch_flush_interval: 10
  initial_tail_lines: 100
  log_files:
    - "/user/logs/swarm_launcher.log"
node_id: "my-node-123"
jwt_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."  # <-- Replace with your actual JWT
```

- **log_files**: List the log files you want to monitor. For RL-Swarm, this should be `/user/logs/swarm_launcher.log` (relative to the `rl-swarm` directory).
- **node_id**: Set a unique identifier for your node.
- **jwt_token**: Obtain your JWT token from the dashboard settings page after authenticating with your Ethereum wallet.

---

## 4. Obtain Your JWT Token

1. Go to the dashboard settings page.
2. Authenticate with your Ethereum wallet.
3. Copy your JWT token and paste it into the `jwt_token` field in `configs/config.yaml`.

---

## 5. Run the Sidecar

Build and run the log monitoring sidecar:

```sh
go run cmd/monitor/main.go
```

Or build a binary:

```sh
go build -o sidecar cmd/monitor/main.go
./sidecar
```

---

## 6. (Optional) Docker Usage

You can also use Docker to run the sidecar. Make sure your `configs/config.yaml` is mounted into the container.

```sh
docker build -t gswarm-sidecar .
docker run -v $(pwd)/configs/config.yaml:/app/configs/config.yaml gswarm-sidecar
```

---

## 7. Troubleshooting

- Ensure the log files exist and are readable by the sidecar process.
- Check your JWT token is valid and not expired.
- Review logs for errors if the sidecar is not forwarding logs as expected.

---

## 8. Need Help?

Open an issue or check the [README.md](../README.md) for more information.

---

## 9. Down Detector & Telegram Alerting

The sidecar includes a built-in **Down Detector** that can alert you via Telegram if your node appears to be offline or unresponsive.

### How It Works
- The sidecar monitors your log files for new activity.
- If no new log lines are detected for a configurable period (e.g., 5 minutes), the sidecar will send an alert to your Telegram via your configured bot.
- Alerts are only sent if the node goes down after being up (no alert spam on startup or repeated alerts).
- Once the node resumes activity, the alert state resets, and you will be notified again only if it goes down in the future.

### Configuration
Add a `telegram` section to your `configs/config.yaml`:

```yaml
telegram:
  bot_token:    "<your-telegram-bot-token>"   # Get this from @BotFather
  chat_id:      "<your-chat-or-channel-id>"   # Use your user or group/channel ID
  alert_on_down: true                         # Set to true to enable alerts
  down_alert_delay: 300                       # Seconds to wait before alerting (e.g., 300 = 5 minutes)
```

- **bot_token**: Create a Telegram bot with [@BotFather](https://t.me/BotFather) and copy the token.
- **chat_id**: Use your user, group, or channel ID. You can get your user ID from [@userinfobot](https://t.me/userinfobot) or add the bot to a group/channel and use its ID.
- **alert_on_down**: Set to `true` to enable down alerts.
- **down_alert_delay**: How long (in seconds) the node must be inactive before an alert is sent.

### Best Practices
- Set a reasonable `down_alert_delay` to avoid false positives (e.g., 300 seconds).
- Make sure your bot has permission to message you or your group/channel.
- Only one alert is sent per downtime event; you will not be spammed with repeated messages.
- Alerts will not be sent immediately on startup if the node is already down.

### Example Alert
```
[gswarm-sidecar] ALERT: Node 'my-node-123' appears DOWN. No log activity for 5m.
```

---

For more help, open an issue or check the main README.
