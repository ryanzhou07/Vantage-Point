import { AuthFrame } from '../features/auth/AuthFrame';
import { AuthPanel } from '../features/auth/AuthPanel';

export default function Home() {
  return (
    <AuthFrame>
      <AuthPanel />
    </AuthFrame>
  );
}
