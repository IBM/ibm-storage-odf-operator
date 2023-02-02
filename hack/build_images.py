from datetime import datetime
from shutil import rmtree
import os
import git
import subprocess

import yaml
from yaml.loader import SafeLoader


def build_and_push_operator_image(docker_registry, git_branch,  driver_image, console_image, platform, output_file):
    git_url = 'https://github.com/IBM/ibm-storage-odf-operator.git'
    docker_repo_name = 'ibm-storage-odf-operator'
    local_dir = 'ibm-storage-odf-operator'

    print("pre-cleanup")
    if os.path.exists(local_dir):
        rmtree(local_dir)

    print("Cloning git project")
    git.Repo.clone_from(git_url, local_dir, branch=git_branch)

    print("Generating image tag")
    image_tag = generate_image_tag(git_branch)

    print("Building image")
    env_vars = dict(IMAGE_REGISTRY=docker_registry, IMAGE_TAG=image_tag, **os.environ)
    subprocess.run(["make", "-C", f'{local_dir}/', "build"], env=env_vars)
    subprocess.run(["make", "-C", f'{local_dir}/', "docker-build"], env=env_vars)
    subprocess.run(["make", "-C", f'{local_dir}/', "bundle-build"], env=env_vars)
    edit_operator_csv(local_dir, driver_image, console_image)
    subprocess.run(["make", "-C", f'{local_dir}/', "catalog-build"], env=env_vars)

    print("Writing image name to images file")
    image_name = docker_registry + '/' + docker_repo_name + ':' + image_tag
    with open(output_file, "a") as images_file:
        images_file.write(image_name + "\n")

    print("post-cleanup")
    if os.path.exists(local_dir):
        rmtree(local_dir)

    return image_name


def edit_operator_csv(operator_local_dir, driver_image, console_image):
    csv_path = f'{operator_local_dir}/bundle/manifests/ibm-storage-odf-operator.clusterserviceversion.yaml'

    with open(csv_path) as f:
        csv = yaml.load(f, Loader=SafeLoader)
        csv["spec"]["install"]["spec"]["deployments"][1]["spec"]["template"]["spec"]["containers"][0]["image"] = console_image
        csv["spec"]["install"]["spec"]["deployments"][0]["spec"]["template"]["spec"]["containers"][1]["env"][0]["value"] = driver_image

    with open(csv_path, mode="w") as f:
        yaml.dump(csv, f)


def build_and_push_driver_image(docker_registry, git_branch, platform, output_file):
    driver_git_url = 'https://github.com/IBM/ibm-storage-odf-block-driver.git'
    driver_docker_repo = 'ibm-storage-odf-block-driver'
    driver_local_dir = 'ibm-storage-odf-block-driver'
    build_and_push_image(docker_registry, driver_docker_repo, driver_git_url, git_branch, driver_local_dir, platform, output_file)


def build_and_push_console_image(docker_registry, git_branch, platform, output_file):
    console_git_url = 'https://github.com/IBM/ibm-storage-odf-console.git'
    console_docker_repo = 'ibm-storage-odf-plugin'
    console_local_dir = 'ibm-storage-odf-console'
    build_and_push_image(docker_registry, console_docker_repo, console_git_url, git_branch, console_local_dir, platform, output_file)


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
    :return: created image name
    """

    print("pre-cleanup", flush=True)
    if os.path.exists(local_dir):
        rmtree(local_dir)

    print("Cloning git project", flush=True)
    git.Repo.clone_from(git_url, local_dir, branch=git_branch)

    print("Generating image tag", flush=True)
    image_tag = generate_image_tag(git_branch)

    print("Building image", flush=True)
    subprocess.run(["make", "-C", f'{local_dir}/', "build-image",
                    f"REGISTRY={docker_registry}",
                    f"IMAGE_TAG={image_tag}",
                    f"TARGET_BRANCH={git_branch}",
                    f"PLATFORM={platform}"])

    print("Writing image name to images file", flush=True)
    image_name = docker_registry + '/' + docker_repo_name + ':' + image_tag
    with open(output_file, "a") as images_file:
        images_file.write(image_name + "\n")

    print("post-cleanup", flush=True)
    if os.path.exists(local_dir):
        rmtree(local_dir)

    return image_name


def generate_image_tag(branch_name):
    now = datetime.now()
    generated_dt = now.strftime("%d.%m.%y-%H.%M")
    generated_branch_name = branch_name.split("/")[-1]
    image_tag = (generated_dt + "-" + generated_branch_name)[:128]
    return image_tag
