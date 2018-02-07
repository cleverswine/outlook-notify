# Outlook Event Notifier (for Linux)

This application uses OpenID Connect to fetch an offline token (token + refresh token). On initial run, you will have to visit http://localhost:5500 and then log in to your MS account. After that, the token is stored locally (**not securely** in this version).

Notifications are sent using the linux command `notify-send`.

Tested on Linux Mint 18.3.

## Flags

```text
  -client string
        A client that is registered in MS AS with appropraite permissions
  -debug
        enable verbose logging
  -http string
        host:port to use for this application's http server (default "localhost:5500")
  -icon string
        Icon to use for notifications (default "/usr/share/icons/Mint-X-Dark/status/24/stock_appointment-reminder.png")
  -lookahead int
        Minutes of lookahead data to get from calendar (default 60)
  -secret string
        The client secret
  -tenant string
        The MS directory to use for login (default "common")
  -ticker int
        Frequency of reminder checks in seconds (default 30)
  -timeformat string
        Display format for reminder times (default "3:04PM")
```

## Example

```bash
./outlook-notify -client "my-client" -secret "my-secret" -tenant "my-AD-tenant"
```