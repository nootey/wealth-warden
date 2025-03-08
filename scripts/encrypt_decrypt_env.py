import os
import sys
import base64
import hashlib

try:
    from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
    from cryptography.hazmat.primitives import padding, hashes
    from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
    from cryptography.hazmat.backends import default_backend
except ImportError:
    print("The 'cryptography' package is not installed. To install it, run 'pip install cryptography' in your terminal.")
    sys.exit(1)

# Define paths for the secrets file and the env file
SECRETS_FILE = os.path.join('pkg', 'config', '.env.secret')
ENV_FILE = os.path.join('pkg', 'config', '.env')

def get_secrets():
    if os.path.exists(SECRETS_FILE):
        salt = None
        stored_pass = None
        with open(SECRETS_FILE, 'rb') as f:
            for line in f:
                line = line.strip()
                if line.startswith(b'SALT='):
                    salt_str = line[len(b'SALT='):]
                    try:
                        salt = base64.b64decode(salt_str)
                        if not salt:
                            print("Error: The salt in the secrets file is empty!")
                            sys.exit(1)
                    except Exception as e:
                        print("Error decoding salt from file:", e)
                        sys.exit(1)
                elif line.startswith(b'PASS='):
                    stored_pass = line[len(b'PASS='):].strip().decode('utf-8')
        if salt is None:
            print("Error: Salt not found in secrets file!")
            sys.exit(1)
        if stored_pass is None:
            print("Error: PASS not found in secrets file! Please set the password once.")
            sys.exit(1)
        return salt, stored_pass
    else:
        print("Secrets file does not exist!")
        sys.exit(1)

def verify_password(user_password, stored_pass):
    hashed = hashlib.sha256(user_password.encode()).hexdigest()
    return hashed == hashlib.sha256(stored_pass.encode()).hexdigest()

def derive_key(password: bytes, salt: bytes) -> bytes:
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=100000,
        backend=default_backend()
    )
    return kdf.derive(password)

def encrypt_file(password):
    salt, stored_pass = get_secrets()
    if password != stored_pass:
        print("Error: Incorrect password!")
        sys.exit(1)
    key = derive_key(password.encode(), salt)
    iv = os.urandom(16)
    cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
    encryptor = cipher.encryptor()

    # Read the env file and check its header.
    with open(ENV_FILE, 'rb') as f:
        first_line = f.readline().strip()
        if first_line == b'DECRYPTED':
            data = f.read()
        elif first_line == b'ENCRYPTED':
            print("The environment file is already encrypted! Aborting encryption.")
            sys.exit(1)
        else:
            print("Error: The environment file does not have a valid header. It may be corrupted!")
            sys.exit(1)

    padder = padding.PKCS7(128).padder()
    padded_data = padder.update(data) + padder.finalize()
    ciphertext = encryptor.update(padded_data) + encryptor.finalize()

    with open(ENV_FILE, 'wb') as f:
        # Write header, IV, and ciphertext (base64 encoded)
        f.write(b'ENCRYPTED\n')
        f.write(base64.b64encode(iv) + b'\n')
        f.write(base64.b64encode(ciphertext))
    print(f"File encrypted to {ENV_FILE}")

def decrypt_file(password):
    salt, stored_pass = get_secrets()
    if password != stored_pass:
        print("Error: Incorrect password!")
        sys.exit(1)
    key = derive_key(password.encode(), salt)
    with open(ENV_FILE, 'rb') as f:
        header = f.readline().strip()
        if header != b'ENCRYPTED':
            print("Error: File is not encrypted!")
            sys.exit(1)
        iv_b64 = f.readline().strip()
        ct_b64 = f.read()

    iv = base64.b64decode(iv_b64)
    ciphertext = base64.b64decode(ct_b64)

    cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
    decryptor = cipher.decryptor()
    try:
        padded_data = decryptor.update(ciphertext) + decryptor.finalize()
        unpadder = padding.PKCS7(128).unpadder()
        data = unpadder.update(padded_data) + unpadder.finalize()
    except Exception:
        print("Invalid passphrase or file corrupted!")
        sys.exit(1)

    with open(ENV_FILE, 'wb') as f:
        f.write(b'DECRYPTED\n')
        f.write(data)
    print(f"File decrypted to {ENV_FILE}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python encrypt_decrypt_env.py [encrypt|decrypt]")
        sys.exit(1)
    mode = sys.argv[1]

    user_password = input("Enter password: ")

    if mode in ("encrypt", "e"):
        encrypt_file(user_password)
    elif mode in ("decrypt", "d"):
        decrypt_file(user_password)
    else:
        print("Unknown mode. Use 'encrypt' or 'decrypt'.")
