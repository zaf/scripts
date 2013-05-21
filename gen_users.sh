#!/usr/bin/env bash
# Generate SIP user accounts for asterisk config files
# by Lefteris Zafiris

echo -e "[myusers](!)\ntype=friend\ncontext=mypbx\nhost=dynamic\ndisallow=all\nallow=gsm,alaw,ulaw\n"

for i in `seq -f %02.0f 1 60`; do
	echo -e "[user$i](myusers)\npassword=`pwgen -sn1 12`\ncallerid=\"user$i <23222120$i>\"\n"
done
