# Build

download `RSS youtube feed` folder

```
cd RSS youtube feed
go build main.go
```

# Usage

create new ssl key using:
> create_cert/create_cert.go

or
> add key pair manually in lib/Request Settings.go/setupTLS


##
```
main.exe -help
```

```
  -addRss string
        add -addRss=URL to get channelId from URL and if laso -opml was given then channelId will be saved to that opml file
  -hrs int
        add -hrs=HOURS_NUMBER to get urls not older than that number of hours.
        If HOURS_NUMBER equals 0 then UnixTime time from file last_update_UnixTime.txt will be used as not older than time and if all are fetched successfuly UnixTime time now will overwrite old time in last_update_UnixTime.txt (default 168)
  -ignore404 bool
        add -ignore404=true to ignore urls from opml which are can not be reached
  -openall bool
        add -openall=true to get urls from opml instead of from feeds.txt
  -opml string
        add -opml=FILE_NAME.opml to get urls from opml instead of from feeds.txt
  -revertDate bool
        add -revertDate=true to revert to date previous of last request
```
ex.:
```
main.exe -hrs=0 -openall=true -opml=file.opml -ignore404=false
```

##
to add new link to file.opml use:
>  main.exe -**addRss**=https://www.youtube.com/@CHANNEL_NAME -opml=file.opml


##
to go back to previous date use:
>  main.exe -**revert**


# License
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License v3.0 as attached.

“The GNU Affero General Public License requires the operator of a network server to provide the source code of the modified version running there to the users of that server. Therefore, public use of a modified version, on a publicly accessible server, gives the public access to the source code of the modified version.”
