locals {
  context_path     = abspath("${path.root}/../../../${var.build_context != "." ? var.build_context : ""}")
  original_command = join(" ", compact([data.external.inspect_base_image.result.entrypoint, data.external.inspect_base_image.result.cmd]))
  image_id         = var.image_id == null ? docker_image.base_service.image_id : var.image_id
  build_trigger    = var.image_id == null ? sha1(join("", [for f in fileset(local.context_path, "**") : filesha1("${local.context_path}/${f}")])) : var.image_id
}

resource "docker_image" "base_service" {
  name = var.image_id == null ? "${var.tag}_base_service" : var.image_id

  dynamic "build" {
    for_each = var.image_id == null ? [1] : []
    content {
      builder  = "default"
      platform = var.platform
      # NOTE: This assumes the terraform output is three dirs down from the root of the project
      context    = local.context_path
      dockerfile = "${local.context_path}/${var.dockerfile}"
      tag        = ["${var.tag}:base"]
      build_args = var.args
    }
  }
  
  triggers = {
    build_trigger = local.build_trigger
  }
}

# Extract entrypoint and command using Docker CLI via external data source
data "external" "inspect_base_image" {
  depends_on = [docker_image.base_service]
  program    = ["docker", "inspect", docker_image.base_service.image_id, "--format", "{\"entrypoint\":\"{{join .Config.Entrypoint \" \"}}\",\"cmd\":\"{{join .Config.Cmd \" \"}}\"}"]
}

# Next we want to wrap this image withing a suga service
resource "docker_image" "service" {
  name = var.tag
  build {
    # This doesn't actually matter as we aren't copying in anything relative
    builder  = "default"
    context  = "."
    platform = var.platform
    # Use the wrapped dockerfile here
    dockerfile = "${path.module}/wrapped.dockerfile"
    build_args = merge({
      BASE_IMAGE       = local.image_id
      ORIGINAL_COMMAND = local.original_command
    }, var.args)
    tag = ["${var.tag}:latest"]
  }

  triggers = {
    base_image_id = local.image_id 
  }
}
