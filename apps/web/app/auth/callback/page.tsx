import { AuthFrame } from '../../../features/auth/AuthFrame';
import { CallbackPanel } from '../../../features/auth/CallbackPanel';

export default function Callback() {
  return (
    <AuthFrame>
      <CallbackPanel />
    </AuthFrame>
  );
}
