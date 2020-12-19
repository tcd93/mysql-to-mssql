## 0. Connect to Google SQL (MySQL) cloud instance with SSL enabled 
_pem files are attached in this repo_

## 1. Execute `go run .`
![](pic/go_run.png)

## 2. Insert one row to table
```
INSERT INTO sakila.Staff (first_name, last_name, address_id, email, store_id, username)
SELECT 'Jaden', 'Tang', 1, 'Jaden.Tang@nexcel.vn', 1, 'go';
```

## 4. Receive event
![](pic/recorded_insert.png)
