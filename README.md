# Outlook Reminder Notifications (for Linux)

This application monitors an Office365 Outlook calendar and sends "toast" notifications based on appointment reminder settings.

Notifications are sent using the linux command `notify-send` and appear in the notification area of the taskbar.

Tested on Linux Mint 18.3, but should work on any system/taskbar that responds to `notify-send`.

## Install (WIP)

* Set up an application registration in your MS Directory (any directory) with appropriate permissions ("Calendars.Read", "User.Read", "offline_access")
* Build the application and run it, passing your application client ID and secret (see "Flags" and "Example" below)

## Technology

* [Outlook Calendar REST API](https://msdn.microsoft.com/en-us/office/office365/api/calendar-rest-operations)
* [notify-send](https://ss64.com/bash/notify-send.html)

## Authentication

The application uses [OpenID Connect](https://openid.net/connect/) to fetch an offline token (token + refresh token) on behalf of a user. On initial run, you will have to visit http://localhost:5500/token and then log in to your MS account. After that, the token is stored locally (**not securely** in this version).

## Flags

```text
Usage of ./outlook-notify:
  -port string
      host:port to use for this application's http server (default "5500")
  -tenant string
      The MS directory to use for login (default "common")
  -client string
      A client that is registered in MS AS with appropraite permissions
  -secret string
      The client secret
  -lookahead int
      Minutes of lookahead data to get from calendar (default 60)
  -refresh int
      Frequency of refreshing event data from the Graph API in minutes (default 15)
  -ticker int
      Frequency of reminder checks in seconds (default 30)
  -timeformat string
      Display format for reminder times (default "3:04PM")
  -tz string
      Local time zone (default "America/Los_Angeles")
  -icon string
      Icon to use for notifications (default "/usr/share/icons/gnome/32x32/status/appointment-soon.png")
  -debug
      enable verbose logging
  -dry-run
      show a test notification
  -help
      show help
```

## Example

```bash
./outlook-notify -client "my-client" -secret "my-secret" -tenant "my-AD-tenant" -debug
```