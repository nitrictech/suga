variable "suga" {
  type = object({
    name     = string
    stack_id = string
    content_path = string
    services = map(object({
      actions = list(string)
      identities = map(object({
        exports = map(string)
      }))
    }))
  })
}

