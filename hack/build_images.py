from datetime import datetime
from shutil import rmtree
import os
import git
import subprocess

import yaml
from yaml.loader import SafeLoader


OPERATOR_GIT_URL = 'https://github.com/IBM/ibm-storage-odf-operator.git'
DRIVER_GIT_URL = 'https://github.com/IBM/ibm-storage-odf-block-driver.git'
CONSOLE_GIT_URL = 'https://github.com/IBM/ibm-storage-odf-console.git'

DRIVER_DOCKER_REPO_NAME = 'ibm-storage-odf-block-driver'
CONSOLE_DOCKER_REPO_NAME = 'ibm-storage-odf-plugin'
OPERATOR_DOCKER_REPO_NAME = 'ibm-storage-odf-operator'
CATALOG_DOCKER_REPO_NAME = 'ibm-storage-odf-catalog'

DRIVER_CLONE_LOCAL_DIR = 'ibm-storage-odf-block-driver'
CONSOLE_CLONE_LOCAL_DIR = 'ibm-storage-odf-console'
OPERATOR_CLONE_LOCAL_DIR = 'ibm-storage-odf-operator'

OPERATOR_CSV_PATH = f'{OPERATOR_CLONE_LOCAL_DIR}/bundle/manifests/ibm-storage-odf-operator.clusterserviceversion.yaml'


def build_and_push_operator_image(docker_registry, git_branch,  driver_image, console_image, platform, output_file):
    print("Cloning git project", flush=True)
    git.Repo.clone_from(OPERATOR_GIT_URL, OPERATOR_CLONE_LOCAL_DIR, branch=git_branch)

    print("Generating image tag", flush=True)
    image_tag = generate_image_tag(git_branch)
    driver_image_tag = driver_image.split(":")[-1]
    console_image_tag = console_image.split(":")[-1]

    print("Building image")
    env_vars = dict(IMAGE_REGISTRY=docker_registry, PLATFORM=platform,
        IMAGE_TAG=image_tag, DRIVER_IMAGE_TAG=driver_image_tag, CONSOLE_IMAGE_TAG=console_image_tag, **os.environ)

    run_operator_make_command("build", env_vars)
    run_operator_make_command("docker-build", env_vars)
    run_operator_make_command("bundle-build", env_vars)
    run_operator_make_command("build-catalog", env_vars)

    print("Dumping created image name", flush=True)
    operator_image_name = f'{docker_registry}/{OPERATOR_DOCKER_REPO_NAME}:{image_tag}'
    catalog_image_name = f'{docker_registry}/{CATALOG_DOCKER_REPO_NAME}:{image_tag}'
    with open(output_file, "a") as images_file:
        images_file.write(operator_image_name + "\n")
        images_file.write(catalog_image_name + "\n")


def run_operator_make_command(make_command, env_vars):
    subprocess.run(["make", "-C", f'{OPERATOR_CLONE_LOCAL_DIR}/', make_command], env=env_vars, check=True)


def edit_operator_csv(driver_image, console_image):
    with open(OPERATOR_CSV_PATH) as f:
        csv = yaml.load(f, Loader=SafeLoader)
        csv["spec"]["install"]["spec"]["deployments"][1]["spec"]["template"]["spec"]["containers"][0]["image"] = console_image
        csv["spec"]["install"]["spec"]["deployments"][0]["spec"]["template"]["spec"]["containers"][1]["env"][0]["value"] = driver_image

    with open(OPERATOR_CSV_PATH, mode="w") as f:
        yaml.dump(csv, f)


def build_and_push_driver_image(docker_registry, git_branch, platform, output_file):
    build_and_push_image(docker_registry, DRIVER_DOCKER_REPO_NAME, DRIVER_GIT_URL, git_branch,
     DRIVER_CLONE_LOCAL_DIR, platform, output_file)


def build_and_push_console_image(docker_registry, git_branch, platform, output_file):
    build_and_push_image(docker_registry, CONSOLE_DOCKER_REPO_NAME, CONSOLE_GIT_URL, git_branch,
     CONSOLE_CLONE_LOCAL_DIR, platform, output_file)


def build_and_push_image(docker_registry, docker_repo_name, git_url, git_branch, local_dir, platform, output_file):
    """
    build and push odf console/driver docker image
    :param docker_registry: docker registry name including namespace (if exist) to push the image into. e.g. docker.io/tyichye
    :param docker_repo_name: docker repository name of the image. e.g. ibm-storage-odf-plugin, ibm-storage-odf-block-driver
    :param git_url: git url of the image project
    :param git_branch: branch name to build the image from. e.g. release-1.3.0
    :param local_dir: local directory to clone the git repo into
    :param platform: docker build platform. linux/amd64, linux/ppc64le, linux/s390x
    :param output_file: Jenkins archive file to dumping the generated image name
    """

    print("Cloning git project", flush=True)
    git.Repo.clone_from(git_url, local_dir, branch=git_branch)

    print("Generating image tag", flush=True)
    image_tag = generate_image_tag(git_branch)

    print("Building image", flush=True)
    subprocess.run(["make", "-C", f'{local_dir}/', "build-image",
                    f"REGISTRY={docker_registry}",
                    f"IMAGE_TAG={image_tag}",
                    f"TARGET_BRANCH={git_branch}",
                    f"PLATFORM={platform}"], check=True)

    print("Dumping created image name", flush=True)
    image_name = f'{docker_registry}/{docker_repo_name}:{image_tag}'
    with open(output_file, "a") as images_file:
        images_file.write(image_name + "\n")


def generate_image_tag(branch_name):
    now = datetime.now()
    generated_dt = now.strftime("%d.%m.%y-%H.%M")
    generated_branch_name = branch_name.split("/")[-1]
    image_tag = (generated_dt + "-" + generated_branch_name)[:128]
    return image_tag


def cleanup_env(output_file):
    print("cleanup env", flush=True)
    if os.path.exists(DRIVER_CLONE_LOCAL_DIR):
        rmtree(DRIVER_CLONE_LOCAL_DIR)
    if os.path.exists(CONSOLE_CLONE_LOCAL_DIR):
        rmtree(CONSOLE_CLONE_LOCAL_DIR)
    if os.path.exists(OPERATOR_CLONE_LOCAL_DIR):
        rmtree(OPERATOR_CLONE_LOCAL_DIR)
    if os.path.exists(output_file):
        os.remove(output_file)