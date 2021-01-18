## 0. Connect to Google SQL (MySQL) cloud instance with SSL enabled 

## 1. Execute `go run .`
![](pic/go_run.png)

## 2. Insert one row to table
```
INSERT INTO sakila.Staff (first_name, last_name, address_id, email, store_id, username)
SELECT 'Johnny', 'Bravo', 1, 'jb@excellent.com', 1, 'go';
```

## 4. Receive event
