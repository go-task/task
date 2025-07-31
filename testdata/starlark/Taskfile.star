version = "3"

def_task(
  name = "gen-foo",
  desc = "generate a foo fighter",
  cmds = [
    "touch foo.txt",
  ],
  sources = [
    "./foo.txt"
  ],
  status = [
    "test 1 = 0"
  ]
)

def_task(
  name = "gen-bar",
  cmds = [
    "touch bar.txt"
  ],
  sources = [
    "./bar.txt"
  ],
  status = [
    "test 1 = 1"
  ]
)

def_task(
  name = "gen-silent-baz",
  silent = True,
  cmds = [
    "touch baz.txt"
  ],
  sources = [
    "./baz.txt"
  ]
)
