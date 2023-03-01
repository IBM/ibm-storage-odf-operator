from datetime import datetime
import os
import git
import subprocess
import pytz


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


def build_and_push_operator_image(docker_registry, git_branch,  driver_image, console_image, platform, output_file):
    print("Cloning git project", flush=True)
    git.Repo.clone_from(OPERATOR_GIT_URL, OPERATOR_CLONE_LOCAL_DIR, branch=git_branch)

    image_tag = generate_image_tag(git_branch)
    driver_image_tag = driver_image.split(":")[-1]
    console_image_tag = console_image.split(":")[-1]

    print("Building image", flush=True)
    env_vars = dict(IMAGE_REGISTRY=docker_registry, PLATFORM=platform,
        IMAGE_TAG=image_tag, DRIVER_IMAGE_TAG=driver_image_tag, CONSOLE_IMAGE_TAG=console_image_tag, **os.environ)

    run_operator_make_command("build", env_vars)
    run_operator_make_command("docker-build", env_vars)
    run_operator_make_command("bundle-build", env_vars)
    run_operator_make_command("build-catalog", env_vars)

    operator_image_name = f'{docker_registry}/{OPERATOR_DOCKER_REPO_NAME}:{image_tag}'
    catalog_image_name = f'{docker_registry}/{CATALOG_DOCKER_REPO_NAME}:{image_tag}'
    writeCreatedImageToOutputFile(output_file, operator_image_name)
    writeCreatedImageToOutputFile(output_file, catalog_image_name)

def run_operator_make_command(make_command, env_vars):
    subprocess.run(["make", "-C", f'{OPERATOR_CLONE_LOCAL_DIR}/', make_command], env=env_vars, check=True)


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

    image_tag = generate_image_tag(git_branch)

    print("Building image", flush=True)
    subprocess.run(["make", "-C", f'{local_dir}/', "push-image",
                    f"REGISTRY={docker_registry}",
                    f"IMAGE_TAG={image_tag}",
                    f"TARGET_BRANCH={git_branch}",
                    f"PLATFORM={platform}"], check=True)

    image_name = f'{docker_registry}/{docker_repo_name}:{image_tag}'
    writeCreatedImageToOutputFile(output_file, image_name)


def generate_image_tag(branch_name):
    print("Generating image tag", flush=True)
    israelTimeZone = pytz.timezone("Asia/Jerusalem")
    now = datetime.now(israelTimeZone)
    generated_dt = now.strftime("%d.%m.%y-%H.%M")
    generated_branch_name = branch_name.split("/")[-1]
    image_tag = (generated_dt + "-" + generated_branch_name)[:128]
    return image_tag

def writeCreatedImageToOutputFile(output_file, image_name):
    print("Dumping created image name", flush=True)
    with open(output_file, "a") as images_file:
        images_file.write(image_name + "\n")
