-- Project config for LazyVim, loaded by lazy.nvim from the working directory.
--
-- Every page in docs/ has a copy in template/docs/, so Marksman reports links
-- such as structure.md as ambiguous, though they resolve relative to the page.
-- Drop those diagnostics and keep the rest.
return {
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        marksman = {
          handlers = {
            ["textDocument/publishDiagnostics"] = function(err, result, ctx)
              if result then
                result.diagnostics = vim.tbl_filter(function(d)
                  return not d.message:match("^Ambiguous link")
                end, result.diagnostics)
              end
              return vim.lsp.diagnostic.on_publish_diagnostics(err, result, ctx)
            end,
          },
        },
      },
    },
  },
}
