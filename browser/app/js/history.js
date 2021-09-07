import createHistory from "history/createBrowserHistory"
import { iposBrowserPrefix } from "./constants"

const history = createHistory({
  basename: iposBrowserPrefix
})

export default history
