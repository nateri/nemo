package main

import (
	//"errors"
	"bufio"
	"fmt"

	"io"
	"os"
	"strings"

	"encoding/csv"
	"sort"

	//"net/mail"
	//"encoding/json"
	"strconv"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/kennygrant/sanitize"
	"github.com/nateri/eazye"
	"github.com/nateri/go-email/email"
)

const _dateFormat = "02-Jan-2006"

type Args struct {
	StoreInfo     bool `arg:"-b,--info" help:"Store all Email Metadata" default:"false"`
	StoreText     bool `arg:"-x,--text" help:"Store all Emails as Text" default:"false"`
	StoreHtml     bool `arg:"-z,--html" help:"Store all Emails as HTML" default:"false"`
	StoreAttached bool `arg:"-a,--attached" help:"Store all Email Attachments" default:"false"`

	StoreFromList bool `arg:"-f,--from-list" help:"Store all From addresses" default:"false"`

	ScanUid  *ScanUidCmd  `arg:"subcommand:scanuid"`
	ScanDate *ScanDateCmd `arg:"subcommand:scandate"`
}

type ScanUidCmd struct {
	Host   string `arg:"-o,--host" help:"IMAP Host" default:"imap.gmail.com:993"`
	User   string `arg:"required,-u,--user" help:"IMAP User (required)"`
	Pass   string `arg:"required,-p,--pass" help:"IMAP Password (required)"`
	Folder string `arg:"-d,--folder" help:"IMAP Folder" default:"[Gmail]/All Mail"`

	FirstUid    uint64 `arg:"-i,--first" help:"First UID in scan, must be greater than 0" default:"1"`
	MaxNumUids  uint64 `arg:"-t,--num" help:"Max number of UIDs to scan, must be >0" default:"1"`
	UidsPerPage uint64 `arg:"-w,--page" help:"Max number of UIDs to fetch, must be >0" default:"100"`
}

type ScanDateCmd struct {
	Host   string `arg:"-o,--host" help:"IMAP Host" default:"imap.gmail.com:993"`
	User   string `arg:"required,-u,--user" help:"IMAP User (required)"`
	Pass   string `arg:"required,-p,--pass" help:"IMAP Password (required)"`
	Folder string `arg:"-d,--folder" help:"IMAP Folder" default:"[Gmail]/All Mail"`

	FirstDate   time.Time `arg:"-i,--date" help:"First Date in scan, must be on or after 01-JAN-1930" default:"01-JAN-2022"`
	MaxNumDays  uint64    `arg:"-t,--num" help:"Max number of Days to scan, must be >0" default:"100"`
	UidsPerPage uint64    `arg:"-w,--page" help:"Max number of UIDs to fetch, must be >0" default:"100"`
}

func (Args) Version() string {
	return "nemo 0.1.5"
}

var fFromList *os.File
var lFrom map[string][]string
var args Args

func main() {

	/*
		p, err := arg.NewParser(arg.Config{
			//IgnoreEnv: true,
			//IgnoreDefault: true,
		}, &args)
		if nil != err {
			fmt.Println("error creating parser:", err.Error())
			return
		}

		if err := p.Parse(os.Args); nil != err {
			fmt.Println("error parsing arguments:", err.Error())
			return
		}
	*/

	p := arg.MustParse(&args)
	//if err := p.Parse(os.Args); nil != err {
	//	fmt.Println("error parsing arguments:", err.Error())
	//	return
	//}
	if nil == p {
		fmt.Println("Please check arguments")
		return
	}

	var host, folder, user, pass string

	// Validate Commands & Params
	switch {
	case args.ScanUid != nil:
		if args.ScanUid.FirstUid == 0 {
			p.Fail("FirstUid must not be 0")
		}
		if args.ScanUid.MaxNumUids == 0 {
			p.Fail("MaxNumUids must not be 0")
		}
		if args.ScanUid.UidsPerPage == 0 {
			p.Fail("UidsPerPage must not be 0")
		}
		host = args.ScanUid.Host
		folder = args.ScanUid.Folder
		user = args.ScanUid.User
		pass = args.ScanUid.Pass
	case args.ScanDate != nil:
		p.Fail("Todo: Command not supported")
	default:
		p.Fail("No commands to run, try 'nemo -h'")
	}

	if "" == host {
		p.Fail("Invalid Host")
	}
	if "" == folder {
		p.Fail("Invalid Folder")
	}
	if "" == user {
		p.Fail("Invalid User")
	}
	if "" == pass {
		p.Fail("Invalid Password")
	}

	fmt.Println("IMAP Host:", host)
	fmt.Println("IMAP Folder:", folder)

	// @TODO: Read user/pass from file
	// Dev short-cut until file support is added [-u n -p r]
	if "n" == user {
		user = "example@gmail.com"
	}
	if "r" == pass {
		pass = "password"
	}

	info := eazye.MailboxInfo{}
	info.Host = host
	info.TLS = true
	info.User = user
	info.Pwd = pass
	info.Folder = folder

	if err := eazye.ValidateMailboxInfo(info); nil != err {
		fmt.Println("Invalid Login: ", err.Error())
		return
	}

	switch {
	case args.ScanUid != nil:
		initStores(p)
		iterateImapFolder(info)
		if nil != fFromList {
			defer fFromList.Close()
		}

	case args.ScanDate != nil:
		p.Fail("Todo")
	}
}

func initStores(p *arg.Parser) {

	fFromList = nil
	lFrom = nil
	if args.StoreFromList {
		var err error
		fFromList, err = os.OpenFile("froms.csv", os.O_RDWR|os.O_CREATE, 0666)
		//fFromList, err = os.OpenFile("froms.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Println("error opening file for writing From list:", err)
			fFromList = nil
		}
		lFrom = make(map[string][]string)

		// Read lines from file
		csvReader := csv.NewReader(fFromList)
		data, err := csvReader.ReadAll()
		if err != nil {
			fmt.Println("error parsing From list as csv:", err)
			return
		}

		// Deserialize email addresses into lFrom from fFromList
		for i, record := range data {
			for r, e := range record[1:] {
				record[r+1] = "\"\"" + e + "\"\""
			}
			lFrom[record[0]] = record[1:]
			fmt.Println("i:", i, ", line:", lFrom[record[0]])
		}
		csvReader = nil
	}
}

func iterateImapFolderByDate(info eazye.MailboxInfo) {
	// @TODO: Use dates to determine start/stop
	//  and refactor features so both scans can re-use them

	//day := -5940

	//loc, _ := time.LoadLocation("America/Los_Angeles")
	//loc := time.UTC
	//t := time.Now().In(loc)
	//t := time.Now()
	//hours := 3 * 24
	days := 0
	//since := t.Add(-(hours * time.Hour))
	since := time.Now().AddDate(0, 0, days)
	//since := t
	sinceStr := since.Format(_dateFormat)
	fmt.Println("Emails since:", since, ",", sinceStr)
	//emails, err := []Email{Email{}}, errors.New("asdf")
	/*
		emails, err := GetSince(info, since, false, false)
		if err != nil {
			fmt.Println("exiting due to error: ", err)
			return
		}
		for _, e := range emails {
			fmt.Println(e)
		}
	*/

	/*
	   		day -= 1
	   		since := time.Now().AddDate(0, 0, day)
	   		sinceStr := since.Format(_dateFormat)
	   		fmt.Print(fmt.Sprintf(`++++++++++++++++++++++++++
	   On:          %s
	   ++++++++++++++++++++++++++
	   `, sinceStr))

	   		ch, err := GenerateOn(info, since, false, false)
	   		if err != nil {
	   			fmt.Println("exiting due to error: ", err.Error())
	   			return
	   		}
	*/

	ch, err := eazye.GenerateSince(info, since, false, false)
	//ch, err := GenerateAll(info, false, false)
	if err != nil {
		fmt.Println("exiting due to error: ", err.Error())
		return
	}

	i := 0
	for e := range ch {
		i += 1
		if e.Err != nil {
			fmt.Println("error on email", i, ":", e.Err.Error())
			continue
		}
		fmt.Println("Id:            ", i)
		fmt.Println(e.Email.String(true))
		//j, err := json.Marshal("{id: \"", i, "\", "date":", e.Email.InternalDate, ",", e.Email.From, ",", e.Email.Subject")

		//fmt.Println()
	}

}

func iterateImapFolder(info eazye.MailboxInfo) {
	status, err := eazye.GetMailboxStatus(info)
	if err != nil {
		fmt.Println("error retrieving Mailbox Status:", err.Error())
	}
	fmt.Println("UIDNext:            ", status.UIDNext)

	done := true
	var uid uint64 = args.ScanUid.FirstUid - 1
	var page uint32 = 0
	var total uint64 = 0
	var max uint64 = args.ScanUid.MaxNumUids

	if max > args.ScanUid.UidsPerPage {
		done = false
	}
	for {
		page += 1
		uid += 1
		first := uid
		uid += args.ScanUid.UidsPerPage - 1
		if uid > (args.ScanUid.FirstUid + max) {
			uid = args.ScanUid.FirstUid + max - 1
		}
		last := uid
		if (last - args.ScanUid.FirstUid + 1) >= max {
			done = true
		}
		if uint32(last) >= status.UIDNext {
			done = true
		}

		strFirst := strconv.FormatUint(first, 10)
		strLast := strconv.FormatUint(last, 10)

		fmt.Print(fmt.Sprintf(
			`++++++++++++++++++++++++++
Page:               %d
Results:            %d
From UID:           %s
To UID:             %s
++++++++++++++++++++++++++
`, page, total, strFirst, strLast))

		ch, err := eazye.GenerateBetween(info, first, last, false, false)
		if err != nil {
			fmt.Println("exiting due to error: ", err.Error())
			return
		}

		i := 0
		for e := range ch {
			i += 1
			total += 1
			if e.Err != nil {
				fmt.Println("error on email", i, ":", e.Err.Error())

				//if e.Err.Error() == "unable to parse email: mime: no media type" {}
				continue
			}

			fmt.Println("Id:            ", i)
			fmt.Println(e.Email.String(true))

			if args.StoreInfo || args.StoreText || args.StoreHtml || args.StoreAttached {
				strUid := fmt.Sprint(e.Email.UID)
				strName := sanitize.Name(e.Email.Subject)
				strName = strings.Replace(strName, ".", "-", -1)
				strAddr := strings.Replace(e.Email.From.Address, "@", ",", -1)
				strTime := e.Email.InternalDate.Format("2006.01.02-15.04.05")
				strDir := strUid + "," + strAddr + "," + strTime + "," + strName
				strDir = "emails/" + strDir

				err := os.Mkdir("emails", 0755)
				if err != nil {
				}
				err = os.Mkdir(strDir, 0755)
				if err != nil {
				}

				if args.StoreInfo {
					var info []string
					info = append(info, strconv.FormatUint(uint64(e.Email.UID), 10))
					info = append(info, e.Email.From.Address+", \""+e.Email.From.Name+"\"")
					//info = append(info, e.Email.InternalDate.Local().Format("2006-01-02 15:04:05"))
					info = append(info, e.Email.InternalDate.Format("2006-01-02 15:04:05"))
					info = append(info, e.Email.Subject)
					for to := range e.Email.To {
						info = append(info, e.Email.To[to].String())
					}

					file, err := os.OpenFile("./"+strDir+"/info.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
					if nil == err {
						datawriter := bufio.NewWriter(file)

						for _, infoEntry := range info {
							_, _ = datawriter.WriteString(infoEntry + "\n")
						}

						datawriter.Flush()
						file.Close()

					} else {
						fmt.Println("Error Storing Email", fmt.Sprint(e.Email.UID), "to Info file:", err.Error())
					}
				}

				if args.StoreText || args.StoreHtml || args.StoreAttached {
					parsedMessage, err := email.ParseMessage(e.Email.Message.Body)

					if args.StoreText {
						if nil == err {
							messages := parsedMessage.MessagesContentTypePrefix("text/plain")
							for i, m := range messages {
								fmt.Println("Email ", e.Email.UID, " has Text: ", i)
								if err := os.WriteFile("./"+strDir+"/body-"+fmt.Sprint(i)+".txt", m.Body, 0666); err != nil {
									fmt.Println("Error Storing Email", fmt.Sprint(e.Email.UID), "(Part:", i, ") to Text file:", err.Error())
								}
							}
						}
					}
					if args.StoreHtml {
						if nil == err {
							messages := parsedMessage.MessagesContentTypePrefix("text/html")
							for i, m := range messages {
								fmt.Println("Email ", e.Email.UID, " has Html: ", i)
								if err := os.WriteFile("./"+strDir+"/body-"+fmt.Sprint(i)+".html", m.Body, 0666); err != nil {
									fmt.Println("Error Storing Email", fmt.Sprint(e.Email.UID), "(Part:", i, ") to HTML file:", err.Error())
								}
							}
						}
					}
					if args.StoreAttached {
						if nil == err {
							fmt.Println("Email ", e.Email.UID, " has Parts: ", parsedMessage.HasParts())

							messages := parsedMessage.MessagesAll()
							for i, _ := range messages {
								fmt.Println("Email ", e.Email.UID, " has part: ", i)
							}
						}
					}
				}
			}

			listUpdated := false
			if nil != fFromList {
				entry, mapContainsKey := lFrom[e.Email.From.Address]
				if !mapContainsKey {
					var names []string
					if len(e.Email.From.Name) > 0 {
						names = append(names, "\"\"\""+e.Email.From.Name+"\"\"\"")
					} else {
						names = append(names, "\"\"\""+"\"\"\"")
					}
					fmt.Println("New Address:     ", e.Email.From.Address, ", Name:", names)

					lFrom[e.Email.From.Address] = names
					listUpdated = true
				} else {
					lenOfNames := len(lFrom[e.Email.From.Address])
					insertIntoStringArray(lFrom[e.Email.From.Address], "\"\"\""+e.Email.From.Name+"\"\"\"")
					if len(lFrom[e.Email.From.Address]) > lenOfNames {
						fmt.Println("New Name Found", lenOfNames, ":     ", e.Email.From.Name)
						listUpdated = true
					}
					fmt.Println("Existing Address:     ", e.Email.From.Address, ", Name:", entry)
				}

				if listUpdated {
					// Serialize lFrom (map) into to fFromList (file)
					fFromList.Seek(0, io.SeekStart)

					for k, v := range lFrom {
						line := k + "," + strings.Join(v, ",")
						fmt.Fprintln(fFromList, line)
					}
				}
			}
		}

		if done {
			return
		}
	}
}

func insertIntoStringArray(ss []string, s string) []string {
	i := sort.SearchStrings(ss, s)
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}
