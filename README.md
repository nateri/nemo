# nemo
Command line utility with some IMAP functionality

Installation 📡
-------
**Go 1.17+**
```bash
go install -v github.com/nateri/nemo@latest
```
**otherwise**
```bash
go get -v github.com/nateri/nemo
```

Usage 💻
-------
```bash
>nemo -h
Usage: nemo [--info] [--text] [--html] [--attached] [--from-list] <command> [<args>]

Options:
  --info, -b             Store all Email Metadata [default: false]
  --text, -x             Store all Emails as Text [default: false]
  --html, -z             Store all Emails as HTML [default: false]
  --attached, -a         Store all Email Attachments [default: false]
  --from-list, -f        Store all From addresses [default: false]
  --help, -h             display this help and exit
  --version              display version and exit

Commands:
  scanuid
  scandate
```


```bash
>nemo scanuid -h
Usage: nemo scanuid [--host HOST] --user USER --pass PASS [--folder FOLDER] [--first FIRST] [--num NUM] [--page PAGE]

Options:
  --host HOST, -o HOST   IMAP Host [default: imap.gmail.com:993]
  --user USER, -u USER   IMAP User (required)
  --pass PASS, -p PASS   IMAP Password (required)
  --folder FOLDER, -d FOLDER
                         IMAP Folder [default: [Gmail]/All Mail]
  --first FIRST, -i FIRST
                         First UID in scan, must be greater than 0 [default: 1]
  --num NUM, -t NUM      Max number of UIDs to scan, must be >0 [default: 1]
  --page PAGE, -w PAGE   Max number of UIDs to fetch, must be >0 [default: 100]

Global options:
  --info, -b             Store all Email Metadata [default: false]
  --text, -x             Store all Emails as Text [default: false]
  --html, -z             Store all Emails as HTML [default: false]
  --attached, -a         Store all Email Attachments [default: false]
  --from-list, -f        Store all From addresses [default: false]
  --help, -h             display this help and exit
  --version              display version and exit
```

Created with [gonesis](https://github.com/edoardottt/gonesis)
