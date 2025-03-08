import os
import random
import string
import subprocess

def generate_random_key(length):
    if length < 9:
        raise Exception("Minimum number of characters is 8")
    characters = string.ascii_letters + string.digits + string.punctuation
    return ''.join(random.choice(characters) for i in range(length))

def update_env_file(env_file_path, index, value):

    if not os.path.exists(env_file_path):
        raise Exception(f"File not found: {env_file_path}")

    with open(env_file_path, 'r') as f:
        lines = f.readlines()

    index_found = False
    for i, line in enumerate(lines):
        if line.startswith(index+"="):
            lines[i] = f'{index}={value}\n'
            index_found = True

    if not index_found:
        lines.append(f'{index}={value}\n')

    with open(env_file_path, 'w') as f:
        f.writelines(lines)

if __name__ == "__main__":

    env_file_path = os.path.join('pkg', 'config', '.env')
    was_decrypted = False

    # Check if the .env file exists and whether it is encrypted.
    if not os.path.exists(env_file_path):
        print(f".env file not found: {env_file_path}")
        exit(1)

    with open(env_file_path, 'r') as f:
        first_line = f.readline().strip()

    if first_line == "ENCRYPTED":
        print("The .env file is encrypted. Decrypting it...")
        subprocess.run(["python", os.path.join("scripts", "encrypt_decrypt_env.py"), "d"], check=True)
        was_decrypted = True
    else:
        print("The .env file is not encrypted. Proceeding with update.")

    print("Generating frontend sessions keys ...")
    try:
        jwt_web_client_access = generate_random_key(12)
        update_env_file(env_file_path, "JWT_WEB_CLIENT_ACCESS", jwt_web_client_access)
    except Exception as e:
        print("Error for: JWT_WEB_CLIENT_ACCESS", e)

    try:
        jwt_web_client_refresh = generate_random_key(12)
        update_env_file(env_file_path, "JWT_WEB_CLIENT_REFRESH", jwt_web_client_refresh)
    except Exception as e:
        print("Error for: JWT_WEB_CLIENT_REFRESH", e)

    try:
        # 16 bytes is the default value for AES128, if you want higher security,
        # but slightly decreased performance, use different byte values from this template:
        # AES128: 16 - AES192: 24 - AES256 - 32
        jwt_frontend_encode = generate_random_key(32)
        update_env_file(env_file_path, "JWT_WEB_CLIENT_ENCODE_ID", jwt_frontend_encode)
    except Exception as e:
        print("Error for: jwt_web_client_encode", e)

    print("Done")

    if was_decrypted:
        print(".env file was decrypted and needs to be re-encrypted. Re-authenticate")
        subprocess.run(["python", os.path.join("scripts", "encrypt_decrypt_env.py"), "e"], check=True)

    print("Keys generated and written to .env file.")