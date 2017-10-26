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
	domain = flag.String("domain", "", "domain")
	compress = flag.Bool("compress", false, "compress replies")
)

func handleQuery(w dns.ResponseWriter, r *dns.Msg) {

	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = *compress

	switch r.Question[0].Qtype {
	default:
		fallthrough
	case dns.TypeA:

		db, err := sql.Open("mysql", *dbUser + ":" +
			*dbPassword + "@tcp(" +
			*dbHost + ":" + *dbPort + ")/" + *dbName +
			"?parseTime=true")

		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		rows, err := db.Query("SELECT address,expire FROM " +
				*dbTable + " WHERE state=0 AND UPPER(hostname)=? ORDER BY expire DESC",
				strings.ToUpper(strings.TrimSuffix(m.Question[0].Name, ".")))

		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		for rows.Next() {
			var (
				address int
				expire time.Time
				rr  dns.RR
				a   net.IP
				ttl int64
			)

			err := rows.Scan(&address, &expire)
			if err == nil {

				a = int2ip(uint32(address))
				ttl = expire.Unix() - time.Now().Unix()

				rr = &dns.A{
					Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)},
					A:   a.To4(),
				}
			}

			m.Answer = append(m.Answer, rr)
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

func int2ip(nn uint32) net.IP {

	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func main() {

	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	dns.HandleFunc(*domain + ".", handleQuery)
	go serve("tcp")
	go serve("udp")
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}
