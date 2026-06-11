import { render } from 'solid-js/web'
import { Router, Route } from '@solidjs/router'
import App from './App'
import Matches from './pages/Matches'
import Leaderboard from './pages/Leaderboard'
import Notifications from './pages/Notifications'
import './index.css'

render(
  () => (
    <Router root={App}>
      <Route path="/" component={Matches} />
      <Route path="/leaderboard" component={Leaderboard} />
      <Route path="/notifications" component={Notifications} />
    </Router>
  ),
  document.getElementById('root')!
)
