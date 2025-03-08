## Environment

The server uses a .env file to store and read secrets.

Currently, the .env is encrypted and committed. To decrypt it, run the script located in:
`./scripts/encrypt_decrypt_env`. Use arguments e|d to encrypt or decrypt the .env file.

The .env has been encrypted with a salt before it has been committed. Only that salt can decrypt it.

For the script to work, and for the server to be able to read the .env, place a file called `.env.secret` in the same directory,
as the .env file (`./pkg/config`)

This is the desired structure: 

```js
SALT=
PASS=
```