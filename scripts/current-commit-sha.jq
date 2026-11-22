if (.sha | type) == "string" and (.sha | length) > 0 then
  .sha
else
  error("commit response does not contain a SHA")
end
