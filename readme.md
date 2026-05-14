# **Manga Hub**


## Prerequisites

- Go 1.22 or newer  
- (Optional) C compiler 
- Sqlite

---

## 📦 Step 1: Get the Code

Open your terminal and run:

```bash
git clone https://github.com/biubiu2408/MangaHub
cd mangahub
```

You should now be in the `mangahub` directory.

---

## 🔐 Step 2: Configure Environment

Create a `.env` file with your JWT secret:

```powershell
@"
JWT_SECRET=my_super_secret_jwt_key_change_in_production
DB_PATH=./data/mangahub.db

```

## 🗄️ Step 3: Install dependencies

```powershell
@"
go mod tidy

```

## 🗄️ Step 4: Start the server
```powershell
@"
go run ./cmd/api-server

```
The server will now be running! Port used: 8080, 9090, 9091, 9092
a database named "mangahub.db" will be created in your /data directory


## 🗄️ Step 5: Initialize the database
On another terminal, run
```powershell
@"
go run ./data/init.go

```
This will populate the database with mangas & account:

**Expected output:**
```
🗄️  Initializing database...
✅ Manga seed import completed
✅ User seed import completed
✅ Database initialized
```

## 👤 Default User Accounts

| Username | Password | Role | Description |
|----------|----------|------|-------------|
| `admin` | `admin123` | admin | Full access |
| `user` | `user123` | user | Regular user |

⚠️ **Remember to change these passwords for production!**

---


# **How to run CLI - FOR TESTING PURPOSE ONLY, FOR ACTUAL CLIENT SIDE WITH FULL SERVICES, PLEASE CHECKOUT OUR DESKTOP APP**
# **Desktop app: [Mangahub-client](https://github.com/hathucanh13/mangahub-client)**
1. Run this in your terminal 
```bash
go build ./mangahub 
```

2. How to run commands
Available commands:
cd the project folder
```bash
#===== COMMANDS (NO LOGGING IN NEEDED) =====
./mangahub manga list --genre (optional) --title (optional)
./mangahub auth login/signup --username (required) --password (required)

#===== LIBRARY COMMANDS =====
./mangahub library list --status (optional: reading, plan_to_read, completed)
./mangahub library add --manga-id (required, eg: berserk) --status (required) 
./mangahub library remove --manga-id (required, eg: berserk)
./mangahub library update --manga-id --status

#===== UDP NOTIFICATION COMMANDS =====
./mangahub notify register
./mangahub notify subscribe --manga (required) #run in separate terminal
 "Note: If you are running the Mangahub desktop app while testing out this notification feature on CLI, note that because they are all using the udp port 3002, conflict may happen. We recommend testing this feature on either CLI or Desktop App only (On desktop app, try using Admin account for easier testing)"

#===== TCP SYNC COMMAND =====
./mangahub sync connect (try open in 2+ terminals to see live updates!)
./mangahub progress update --manga-id (example: berserk) --chapter 3 #run in separate terminal
./mangahub progress history

#===== GRPC COMMAND =====
./mangahub grpc get --manga (required, eg: berserk) 
./mangahub grpc search --keyword (required, eg: to) --page (optional) --page-size (optional)
./mangahub grpc update-progress --manga-id --chapter #Try this while running sync connect in the other terminal to see live updates!

#===== CHAT COMMAND ======
./mangahub chat join --manga (required, eg: naruto)

#===== ADMIN ONLY COMMAND ======
./mangahub admin update-chapter --manga --chapter (Update new chapter releases)
```
# HAVE FUN AND HAPPY NEW YEAR!


