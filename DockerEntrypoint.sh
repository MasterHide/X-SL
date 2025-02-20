#!/bin/sh

# Start fail2ban
[ $X_UI_ENABLE_FAIL2BAN == "true" ] && fail2ban-client -x start

# Run x-sl
exec /app/x-sl
