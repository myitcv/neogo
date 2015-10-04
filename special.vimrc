syntax on
filetype off
set noswapfile

function! s:RequireGoHost(host)
  try
    let channel_id = rpcstart($HOME.'/.nvim/plugins/go/plugin_host')
    call rpcrequest(channel_id, 'plugin_load', 'go')
    return channel_id
  catch
    echomsg v:exception
  endtry
  throw 'Failed to load Go host'.
endfunction

if has('nvim')
  call remote#host#Register('go', '*', function('s:RequireGoHost'))
  try
    call remote#define#FunctionOnHost('go', 'BufferUpdate', 0, 'BufferUpdate', {})
  catch
    echomsg v:exception
  endtry
endif

silent! colorscheme sahara
au CursorMoved,TextChanged,TextChangedI <buffer> call BufferUpdate()
