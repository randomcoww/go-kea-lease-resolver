// resolver for kea DHCP leases
// based on https://github.com/miekg/exdns/blob/master/reflect/reflect.go
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/miekg/dns"
	"time"
	"strings"
	"strconv"
	"encoding/binary"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var (
	dbHost = flag.String("h", "127.0.0.1", "database host")
	dbName = flag.String("d", "", "database name")
	dbPassword = flag.String("w", "", "database password")
	dbPort = flag.String("p", "3306", "database port")
	dbUser = flag.String("u", "", "database user")
	dbTable = flag.String("t", "lease4", "kea lease table")
	listenPort = flag.String("listen", "53530", "listen port")
	db *sql.DB
)

func handleQuery(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = true

	switch r.Question[0].Qtype {

	default:
		fallthrough

	case dns.TypePTR:
	 	arr := strings.Split(strings.ToUpper(strings.TrimSuffix(m.Question[0].Name, ".in-addr.arpa.")), ".")
		size := len(arr) - 1
		ip := make(net.IP, 4)

		for i := range arr {
			v, _ := strconv.ParseInt(arr[size - i], 10, 16)
			ip[i] = byte(v)
		}

		rows, err := db.Query("SELECT hostname,expire FROM " +
				*dbTable + " WHERE state=0 AND address=? ORDER BY expire DESC",
				ipToInt(ip))

		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		var (
			hostname string
			expire time.Time
		)

		for rows.Next() {
			err := rows.Scan(&hostname, &expire)
			if err == nil {

				rr := &dns.PTR{
					Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: uint32(expire.Unix() - time.Now().Unix())},
					Ptr: hostname + ".",
				}

				m.Answer = append(m.Answer, rr)
			}
		}

	case dns.TypeA:
		rows, err := db.Query("SELECT address,expire FROM " +
				*dbTable + " WHERE state=0 AND UPPER(hostname) IN (?,?) ORDER BY expire DESC",
				strings.ToUpper(m.Question[0].Name),
				strings.ToUpper(strings.TrimSuffix(m.Question[0].Name, ".")))

		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		var (
			address int
			expire time.Time
		)

		for rows.Next() {
			err := rows.Scan(&address, &expire)
			if err == nil {

				rr := &dns.A{
					Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(expire.Unix() - time.Now().Unix())},
					A:   intToIP(uint32(address)).To4(),
				}

				m.Answer = append(m.Answer, rr)
			}
		}
	}

	w.WriteMsg(m)
}


func serve(net string) {
	server := &dns.Server{Addr: ":" + *listenPort, Net: net, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the "+net+" server: %s\n", err.Error())
	}
}


// https://gist.github.com/ammario/649d4c0da650162efd404af23e25b86b
func ipToInt(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip)
}


// https://gist.github.com/ammario/649d4c0da650162efd404af23e25b86b
func intToIP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}


func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()


	var err error

	db, err = sql.Open("mysql", *dbUser + ":" +
		*dbPassword + "@tcp(" +
		*dbHost + ":" + *dbPort + ")/" + *dbName +
		"?parseTime=true")

	if err != nil {
		panic(err.Error())
	}
	defer db.Close()


	dns.HandleFunc(".", handleQuery)
	go serve("tcp")
	go serve("udp")
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}
