import { render } from 'solid-js/web'
import { Router, Route } from '@solidjs/router'
import App from './App'
import Matches from './pages/Matches'
import Leaderboard from './pages/Leaderboard'
import './index.css'

render(
  () => (
    <Router root={App}>
      <Route path="/" component={Matches} />
      <Route path="/leaderboard" component={Leaderboard} />
    </Router>
  ),
  document.getElementById('root')!
)
