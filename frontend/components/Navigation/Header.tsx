import { Link, useLocation } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import { Button } from "../../components/ui/button";

export function Header() {
  const { user, logout } = useAuth();
  const location = useLocation();

  const handleLogout = async () => {
    await logout();
  };

  const isActive = (path: string) => location.pathname === path;

  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-50">
      <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
        <div className="flex items-center space-x-6">
          <Link to="/" className="text-xl font-bold text-gray-900">
            ISOMAP
          </Link>
          <nav className="flex space-x-4">
            <Link
              to="/"
              className={`text-sm font-medium ${
                isActive("/")
                  ? "text-blue-600"
                  : "text-gray-600 hover:text-gray-900"
              }`}
            >
              Home
            </Link>
            <Link
              to="/map"
              className={`text-sm font-medium ${
                isActive("/map")
                  ? "text-blue-600"
                  : "text-gray-600 hover:text-gray-900"
              }`}
            >
              Map
            </Link>
          </nav>
        </div>
        <div className="flex items-center space-x-4">
          <span className="text-sm text-gray-600">
            Signed in as <span className="font-medium">{user?.name}</span>
          </span>
          <Button variant="outline" size="sm" onClick={handleLogout}>
            Sign Out
          </Button>
        </div>
      </div>
    </header>
  );
}
