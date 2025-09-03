variable "suga" {
  type = object({
    name     = string
    stack_id = string
    env_var_key = string
    services = map(object({
      actions = list(string)
      identities = map(object({
        exports = map(string)
      }))
    }))
  })
}
