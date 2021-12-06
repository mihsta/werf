


FROM ghcr.io/werf/builder:afd00fa1348645658b718df6b2b7447c6cead90b@sha256:829a6fd3d5850fc24366b5a3fb10402fe2d5615dcfd045b7225819c7a8ec5011
ARG build_version
ADD cmd /werf/cmd
ADD pkg /werf/pkg
ADD go.mod /werf/go.mod
ADD go.sum /werf/go.sum
ADD scripts/build_release_v3.sh /werf/scripts/build_release_v3.sh
RUN cd /werf && go mod download
RUN cd /werf && ./scripts/build_release_v3.sh $build_version




FROM registry.fedoraproject.org/fedora:latest
ARG build_version
# Don't include container-selinux and remove
# directories used by yum that are just taking
# up space.
RUN useradd build
RUN yum -y update
#RUN rpm --restore shadow-utils 2>/dev/null
#RUN yum -y install buildah xz --exclude container-selinux
RUN yum -y install fuse-overlayfs git
RUN rm -rf /var/cache /var/log/dnf* /var/log/yum.*

RUN yum install -y git curl gpg

#ADD https://raw.githubusercontent.com/containers/buildah/main/contrib/buildahimage/stable/containers.conf /etc/containers/

# Adjust storage.conf to enable Fuse storage.
#RUN chmod 644 /etc/containers/containers.conf; sed -i -e 's|^#mount_program|mount_program|g' -e '/additionalimage.*/a "/var/lib/shared",' -e 's|^mountopt[[:space:]]*=.*$|mountopt = "nodev,fsync=0"|g' /etc/containers/storage.conf
#RUN mkdir -p /var/lib/shared/overlay-images /var/lib/shared/overlay-layers /var/lib/shared/vfs-images /var/lib/shared/vfs-layers; touch /var/lib/shared/overlay-images/images.lock; touch /var/lib/shared/overlay-layers/layers.lock; touch /var/lib/shared/vfs-images/images.lock; touch /var/lib/shared/vfs-layers/layers.lock

# Define uid/gid ranges for our user https://github.com/containers/buildah/issues/3053
RUN echo build:2000:50000 >/etc/subuid && echo build:2000:50000 >/etc/subgid

#FIXME(ilya-lesikov):
RUN mkdir -p /home/build/.local/share/containers
RUN chown -R build:build /home/build

VOLUME /var/lib/containers
VOLUME /home/build/.local/share/containers

# Set an environment variable to default to chroot isolation for RUN
# instructions and "buildah run".
#ENV BUILDAH_ISOLATION=chroot

COPY --from=0 /werf/release-build/$build_version/linux-amd64/bin/werf /usr/local/bin/werf

ENV WERF_CONTAINER_RUNTIME=buildah
