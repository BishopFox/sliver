lua << EOF
require('go').setup({
  test_flags = { '-v', '-count=1' },
  test_env = {GO_LARK_TEST_MODE = 'local'},
  test_popup_width = 120,
  test_open_cmd = 'tabedit',
  tags_options = {'-sort'},
  tags_transform = 'camelcase',
})
EOF
