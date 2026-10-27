# Command to run while running on docker
For the Docker database on port 3307:


Username:  root
Password:  root-local-password
There's also an application user, which the backend actually uses — either works in Workbench:


Username:  rera
Password:  rera-local-password

Inside Docker Desktop, if you'd rather stay there: Containers → rera-mysql → Exec tab, then:

# command to run on docker desktp mysql
mysql -u root -p
PASSWORD - root-local-password
mysql -uroot -proot-local-password rera_properties
SHOW DATABASES;
USE rera_properties;
Then normal SQL — SHOW TABLES;, SELECT * FROM users;. Type exit to leave.


Or from your terminal, no GUI:


docker compose --env-file .env.docker exec mysql \
  mysql -uroot -proot-local-password rera_properties
One-off query without opening a session:


docker compose --env-file .env.docker exec -T mysql \
  mysql -uroot -proot-local-password \
  -e "SELECT id, name, purpose, property_status FROM rera_properties.properties;"
Worth knowing: the container's MySQL and your native MySQL are completely separate databases that happen to share a schema name. Same tables, different data. The container's lives in the rera_mysql_data volume; your native one is in C:\ProgramData\MySQL. When something looks missing, checking which port you're connected to is usually the answer.

# Confirm the Docker MySQL credentials
Input - grep -n "MYSQL_ROOT_PASSWORD\|^DB_USER\|^DB_PASSWORD\|^DB_NAME" .env.docker
output :#     MYSQL_USER and use MYSQL_ROOT_PASSWORD") and the container exits;
26:DB_USER=rera
27:DB_PASSWORD=rera-local-password
28:DB_NAME=rera_properties
29:MYSQL_ROOT_PASSWORD=root-local-password


# Command to run on terminal for docker

Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.IPAddress -notlike "127.*" -and $_.IPAddress -notlike "169.254.*" } | Select-Object IPAddress, InterfaceAlias | Format-Table -AutoSize

->docker version
->docker compose --env-file .env.docker build
->docker compose --env-file .env.docker up -d
->docker compose --env-file .env.docker ps
->docker compose --env-file .env.docker logs -f backend

# 6. Open it

Site → http://localhost:5173
API health → http://localhost:8080/api/v1/health
Sign in → admin@reraproperties.local / Admin@12345

If other devices on your Wi-Fi need to reach it (phone, colleague's laptop), three values must point at 10.1.60.114 instead of localhost — the SPA runs in their browser, so localhost:8080 would mean their machine, and images would break the same way. Say the word and I'll make the change; it needs a frontend rebuild because Vite bakes the API URL in at build time.

# to get the otp from server run
docker compose --env-file .env.docker logs backend | Select-String "verification code" | Select-Object -Last 1
docker compose --env-file .env.docker logs backend 2>&1 | grep -i "verification code" | tail -3

# Check volumes and whether user rows survived
docker volume ls | grep -i rera; echo "=== users in db ==="; docker compose --env-file .env.docker exec -T mysql mysql -uroot -proot-local-password -e "SELECT id, email, mobile, created_at FROM rera_properties.users ORDER BY id DESC LIMIT 10;" 2>&1 | tail -15

# docker compose ... down -v      # ← the -v deletes the volumes

# Use these instead:
docker compose --env-file .env.docker stop     # pause; data kept
docker compose --env-file .env.docker start    # resume
docker compose --env-file .env.docker down     # remove containers; data kept
docker compose --env-file .env.docker up -d    # bring back
docker compose --env-file .env.docker restart backend
docker compose --env-file .env.docker exec -T mysql \
  mysql -uroot -proot-local-password \
  -e "SELECT id, email FROM rera_properties.users ORDER BY id DESC LIMIT 3;"

# Confirm the published MySQL port
docker compose --env-file .env.docker ps --format "table {{.Service}}\t{{.Ports}}" 2>&1 | head -8


MySQL Workbench — open it, click + next to "MySQL Connections":

Field	Value
Connection Name	RERA Docker
Hostname	127.0.0.1
Port	3307
Username	root
Password	root-local-password
Default Schema	rera_properties

# command to run when u want your website visible to every one
docker run --rm --network rera_rera cloudflare/cloudflared:latest tunnel --url http://frontend:80
