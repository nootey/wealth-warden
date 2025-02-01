import os
import random
import string

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

    env_file_path = "./.env"
    print("Generating frontend sessions keys ...")
    try:
        jwt_frontend_access = generate_random_key(12)
        update_env_file(env_file_path, "JWT_SECRET_FRONTEND_ACCESS", jwt_frontend_access)
    except Exception as e:
        print("Error for: jwt_frontend_access", e)

    try:
        jwt_frontend_refresh = generate_random_key(12)
        update_env_file(env_file_path, "JWT_SECRET_FRONTEND_REFRESH", jwt_frontend_refresh)
    except Exception as e:
        print("Error for: jwt_frontend_refresh", e)

    try:
        # 16 bytes is the default value for AES128, if you want higher security,
        # but slightly decreased performance, use different byte values from this template:
        # AES128: 16 - AES192: 24 - AES256 - 32
        jwt_frontend_encode = generate_random_key(32)
        update_env_file(env_file_path, "JWT_FRONTEND_ENCODE_KEY", jwt_frontend_encode)
    except Exception as e:
        print("Error for: jwt_frontend_encode", e)
    print("Done")

    print("Generating api jwt keys ...")
    try:
        api_licence = generate_random_key(12)
        update_env_file(env_file_path, "JWT_SECRET_API", api_licence)
    except Exception as e:
        print("Error for: api_licence", e)
    print("Done")

    print("Keys generated and written to .env file.")