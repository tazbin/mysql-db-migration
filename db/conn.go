package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/crypto/ssh"

	"github.com/go-sql-driver/mysql"
)

// DB holds the database connection pool
var DB *sql.DB

// Config holds SSH and DB credentials
type Config struct {
	SSHUser    string
	SSHHost    string
	SSHPort    int
	SSHKeyPath string
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     int
	DBName     string
}

// Connect initializes the SSH tunnel and DB connection
func Connect(cfg Config) {
	fmt.Println("⏳ Connecting to database...")

	// Read the private key
	key, err := ioutil.ReadFile(cfg.SSHKeyPath)
	if err != nil {
		log.Fatal("Failed to read private key:", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatal("Failed to parse private key:", err)
	}

	// SSH client config
	sshConfig := &ssh.ClientConfig{
		User: cfg.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to SSH
	sshAddr := fmt.Sprintf("%s:%d", cfg.SSHHost, cfg.SSHPort)
	sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		log.Fatal("Failed to dial SSH:", err)
	}

	// Register a custom dialer
	mysql.RegisterDialContext("mysql+ssh", func(ctx context.Context, addr string) (net.Conn, error) {
		return sshClient.Dial("tcp", addr)
	})

	// Build DSN
	dsn := fmt.Sprintf("%s:%s@mysql+ssh(%s:%d)/%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// Open DB
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	// Test connection
	if err := DB.Ping(); err != nil {
		log.Fatal("Failed to ping DB:", err)
	}

	log.Println("✅ Connected to MySQL through SSH tunnel!")
}
