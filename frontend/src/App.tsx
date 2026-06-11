import { ParentComponent } from 'solid-js'
import Nav from './components/Nav'

const App: ParentComponent = (props) => {
  return (
    <>
      <header class="nav-wrapper">
        <Nav />
      </header>
      <div class="app">
        <main class="container">
          {props.children}
        </main>
      </div>
    </>
  )
}

export default App
