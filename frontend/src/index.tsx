/* @refresh reload */
import { render } from 'solid-js/web'
import './index.css'
import App from './App.tsx'
import { installLampFlash } from './lib/lampFlash.ts'

const root = document.getElementById('root')

installLampFlash()
render(() => <App />, root!)
